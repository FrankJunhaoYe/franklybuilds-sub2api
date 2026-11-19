package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *scheduledTestPlanRepository) ClaimQuality(ctx context.Context, p *service.ScheduledTestPlan) (bool, error) {
	// updated_at is also the fencing token: a changed plan or a later claim
	// invalidates the pending result, even if the first worker is still alive.
	row := r.db.QueryRowContext(ctx, `UPDATE scheduled_test_plans SET quality_lease_until=now()+interval '6 minutes',
 updated_at=clock_timestamp() WHERE id=$1 AND enabled AND quality_rule IS NOT NULL
 AND updated_at=$2 AND next_run_at<=now() AND (quality_lease_until IS NULL OR quality_lease_until<=now()) RETURNING updated_at`, p.ID, p.UpdatedAt)
	err := row.Scan(&p.UpdatedAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *scheduledTestPlanRepository) QualityMetrics(ctx context.Context, accountID int64, model string) (service.QualityMetrics, error) {
	var metrics service.QualityMetrics
	// Recorded successes and upstream HTTP errors from the same 60-minute window.
	// An idle/isolated model has no samples; it can still recover by answering probes.
	err := r.db.QueryRowContext(ctx, `WITH successes AS (
 SELECT count(*) AS n,
 CASE WHEN count(first_token_ms) FILTER(WHERE stream AND first_token_ms>=0)>=3 THEN avg(first_token_ms) FILTER(WHERE stream AND first_token_ms>=0) END AS ttft,
 CASE WHEN count(*) FILTER(WHERE output_tokens>0 AND duration_ms>0)>=3 THEN avg(1000.0*output_tokens/nullif(duration_ms,0)) FILTER(WHERE output_tokens>0 AND duration_ms>0) END AS tps
 FROM usage_logs WHERE account_id=$1 AND model=$2 AND created_at>=now()-interval '60 minutes'
 ), errors AS (SELECT count(*) AS n FROM ops_error_logs WHERE account_id=$1 AND model=$2
 AND upstream_status_code BETWEEN 400 AND 599 AND created_at>=now()-interval '60 minutes')
 SELECT ttft,tps,CASE WHEN successes.n+errors.n>=3 THEN 100.0*errors.n/(successes.n+errors.n) END FROM successes,errors`, accountID, model).Scan(&metrics.TTFT, &metrics.TPS, &metrics.ErrorPercent)
	return metrics, err
}

func (r *scheduledTestPlanRepository) FinishQuality(ctx context.Context, p *service.ScheduledTestPlan, result *service.ScheduledTestResult, next time.Time) error {
	state, err := json.Marshal(p.QualityState)
	if err != nil {
		return err
	}
	evidence, err := json.Marshal(result.Evidence)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `UPDATE scheduled_test_plans SET quality_state=$1,quality_lease_until=NULL,
 last_run_at=$2,next_run_at=$3,updated_at=clock_timestamp() WHERE id=$4 AND enabled AND updated_at=$5 AND quality_lease_until>now()`,
		string(state), result.FinishedAt, next, p.ID, p.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil || n == 0 {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO scheduled_test_results(plan_id,status,latency_ms,started_at,finished_at,quality_evidence)
 VALUES($1,$2,$3,$4,$5,$6)`, p.ID, result.Status, result.LatencyMs, result.StartedAt, result.FinishedAt, string(evidence))
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM scheduled_test_results WHERE id IN(SELECT id FROM scheduled_test_results WHERE plan_id=$1 ORDER BY created_at DESC,id DESC OFFSET $2)`, p.ID, p.MaxResults)
	if err != nil {
		return err
	}
	return tx.Commit()
}
