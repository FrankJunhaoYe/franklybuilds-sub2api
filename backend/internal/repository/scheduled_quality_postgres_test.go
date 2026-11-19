//go:build unit

package repository

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScheduledQualityPostgres(t *testing.T) {
	db := scopedAdminKeyTestDB(t)
	ctx := context.Background()
	_, err := db.Exec(`CREATE TABLE accounts(id bigint primary key); INSERT INTO accounts VALUES(1);
 CREATE TABLE scheduler_outbox(event_type text,account_id bigint);
 CREATE TABLE usage_logs(account_id bigint, model text, created_at timestamptz, stream bool, first_token_ms int,output_tokens int,duration_ms int);
 CREATE TABLE ops_error_logs(account_id bigint,model text,created_at timestamptz,upstream_status_code int)`)
	require.NoError(t, err)
	for _, name := range []string{"066_add_scheduled_test_tables.sql", "070_add_scheduled_test_auto_recover.sql", "326_scheduled_account_quality.sql"} {
		migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		require.NoError(t, err)
		_, err = db.Exec(string(migration))
		require.NoError(t, err)
	}
	repo := NewScheduledTestPlanRepository(db)
	results := NewScheduledTestResultRepository(db)
	past := time.Now().Add(-time.Minute)
	plan := &service.ScheduledTestPlan{AccountID: 1, ModelID: "m", CronExpression: "* * * * *", Enabled: true, NextRunAt: &past, MaxResults: 2}
	legacy, err := repo.Create(ctx, plan)
	require.NoError(t, err)
	require.Nil(t, legacy.QualityRule)
	plan.QualityRule = &service.QualityRule{Models: []string{"m"}, Action: "model", Failures: 1, Passes: 1}
	create := func() *service.ScheduledTestPlan { p, err := repo.Create(ctx, plan); require.NoError(t, err); return p }
	claim := func(p *service.ScheduledTestPlan) {
		ok, err := repo.ClaimQuality(ctx, p)
		require.NoError(t, err)
		require.True(t, ok)
	}
	finish := func(p *service.ScheduledTestPlan) {
		p.QualityState = map[string]service.QualityState{"m": {Held: true}}
		now := time.Now()
		require.NoError(t, repo.FinishQuality(ctx, p, &service.ScheduledTestResult{StartedAt: now, FinishedAt: now, Status: "failed", Evidence: []service.QualityEvidence{{Model: "m", Verdict: "failed"}}}, now.Add(time.Hour)))
	}
	held := func() bool {
		var v bool
		require.NoError(t, db.QueryRow("SELECT COALESCE(quality_holds ? 'model:m',false) FROM accounts WHERE id=1").Scan(&v))
		return v
	}
	first, second := create(), create()
	claim(first)
	ok, err := repo.ClaimQuality(ctx, first)
	require.NoError(t, err)
	require.False(t, ok)
	finish(first)
	claim(second)
	finish(second)
	require.True(t, held())
	history, err := results.ListByPlanID(ctx, first.ID, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "m", history[0].Evidence[0].Model)
	require.NoError(t, repo.Delete(ctx, first.ID))
	require.True(t, held())
	second.Enabled = false
	_, err = repo.Update(ctx, second)
	require.NoError(t, err)
	require.False(t, held())
	stale := create()
	claim(stale)
	changed := *stale
	changed.Enabled = false
	_, err = repo.Update(ctx, &changed)
	require.NoError(t, err)
	finish(stale)
	require.False(t, held())
	history, err = results.ListByPlanID(ctx, stale.ID, 10)
	require.NoError(t, err)
	require.Empty(t, history)
	stats, err := repo.QualityMetrics(ctx, 1, "m")
	require.NoError(t, err)
	require.Nil(t, stats.TPS)
	_, err = db.Exec(`INSERT INTO usage_logs SELECT 1,'m',now(),true,40,100,2000 FROM generate_series(1,3);
 INSERT INTO ops_error_logs VALUES(1,'m',now(),502)`)
	require.NoError(t, err)
	stats, err = repo.QualityMetrics(ctx, 1, "m")
	require.NoError(t, err)
	require.Equal(t, 50.0, *stats.TPS)
	require.Equal(t, 25.0, *stats.ErrorPercent)
	require.Equal(t, 40.0, *stats.TTFT)
}
