package service

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

type QualityRule struct {
	Models          []string `json:"models"`
	Action          string   `json:"action"`
	GroupID         int64    `json:"group_id,omitempty"`
	Failures        int      `json:"failures"`
	Passes          int      `json:"passes"`
	MaxTTFT         int      `json:"max_ttft_ms,omitempty"`
	MinTPS          float64  `json:"min_tps,omitempty"`
	MaxErrorPercent float64  `json:"max_error_percent,omitempty"`
}

func (r *QualityRule) validate() error {
	if r == nil {
		return nil
	}
	if len(r.Models) < 1 || len(r.Models) > 8 {
		return fmt.Errorf("select 1 to 8 text models")
	}
	seen := map[string]bool{}
	for _, m := range r.Models {
		if m == "" || len(m) > 100 || strings.TrimSpace(m) != m || seen[m] {
			return fmt.Errorf("invalid or duplicate quality model")
		}
		seen[m] = true
	}
	switch r.Action {
	case "observe", "account", "model":
	case "group":
		if r.GroupID <= 0 {
			return fmt.Errorf("select a group for isolation")
		}
	default:
		return fmt.Errorf("invalid quality action")
	}
	if r.Failures < 1 || r.Failures > 20 || r.Passes < 1 || r.Passes > 20 || r.MaxTTFT < 0 || r.MaxTTFT > 300000 || r.MinTPS < 0 || r.MinTPS > 10000 || r.MaxErrorPercent < 0 || r.MaxErrorPercent > 100 {
		return fmt.Errorf("invalid quality threshold")
	}
	return nil
}

type QualityState struct {
	Failures int  `json:"failures"`
	Passes   int  `json:"passes"`
	Held     bool `json:"held"`
}

type QualityMetrics struct {
	TTFT         *float64 `json:"ttft_ms,omitempty"`
	TPS          *float64 `json:"tps,omitempty"`
	ErrorPercent *float64 `json:"error_percent,omitempty"`
}

type QualityEvidence struct {
	Model    string `json:"model"`
	Question string `json:"question"`
	Expected string `json:"expected"`
	Answer   string `json:"answer"`
	Verdict  string `json:"verdict"`
	Reason   string `json:"reason,omitempty"`
	QualityMetrics
}

func qualityQuestion() (string, string) {
	a, b, c := rand.IntN(83)+13, rand.IntN(13)+3, rand.IntN(37)+5
	switch rand.IntN(3) {
	case 0:
		return fmt.Sprintf("Compute (%d + %d) * %d. Reply with only the integer.", a, c, b), strconv.Itoa((a + c) * b)
	case 1:
		return fmt.Sprintf("Starting with %d tokens, add %d on each odd step and remove %d on each even step. After seven steps, reply with only the final integer.", a, c, b), strconv.Itoa(a + 4*c - 3*b)
	default:
		return fmt.Sprintf("Count the integers from 1 through %d inclusive that are divisible by %d. Reply with only the integer.", a*c, b), strconv.Itoa(a * c / b)
	}
}

func (r QualityRule) judge(e *QualityEvidence, probeOK bool) {
	if !probeOK {
		e.Verdict = "unknown"
		return
	}
	e.Verdict = "failed"
	switch {
	case strings.TrimSpace(e.Answer) != e.Expected:
		e.Reason = "answer_mismatch"
	case r.MaxTTFT > 0 && e.TTFT != nil && *e.TTFT > float64(r.MaxTTFT):
		e.Reason = "ttft"
	case r.MinTPS > 0 && e.TPS != nil && *e.TPS < r.MinTPS:
		e.Reason = "tps"
	case r.MaxErrorPercent > 0 && e.ErrorPercent != nil && *e.ErrorPercent > r.MaxErrorPercent:
		e.Reason = "upstream_errors"
	default:
		e.Verdict = "passed"
	}
}

func (r QualityRule) advance(state QualityState, verdict string) QualityState {
	switch verdict {
	case "failed":
		state.Passes = 0
		state.Failures = min(state.Failures+1, r.Failures)
		if state.Failures >= r.Failures && r.Action != "observe" {
			state.Held = true
		}
	case "passed":
		state.Failures = 0
		state.Passes = min(state.Passes+1, r.Passes)
		if state.Passes >= r.Passes {
			state.Held = false
		}
	default:
		state.Failures = 0
		state.Passes = 0
	}
	return state
}

type qualityGroupContext struct{}

func withQualityGroup(ctx context.Context, id *int64) context.Context {
	if id == nil {
		return ctx
	}
	return context.WithValue(ctx, qualityGroupContext{}, *id)
}

func (s *ScheduledTestRunnerService) runQuality(ctx context.Context, p *ScheduledTestPlan) error {
	ok, err := s.planRepo.ClaimQuality(ctx, p)
	if err != nil || !ok {
		return err
	}
	start := time.Now()
	result := &ScheduledTestResult{PlanID: p.ID, StartedAt: start, Status: "success"}
	if p.QualityState == nil {
		p.QualityState = map[string]QualityState{}
	}
	for _, model := range p.QualityRule.Models {
		if err := ctx.Err(); err != nil {
			return err
		}
		question, expected := qualityQuestion()
		probeCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		sample, err := s.accountTestSvc.runBackground(probeCtx, p.AccountID, model, question)
		cancel()
		if err != nil {
			return err
		}
		e := QualityEvidence{Model: model, Question: question, Expected: expected, Answer: sample.ResponseText}
		if sample.Status != "success" {
			e.Reason = "probe_unavailable"
		}
		if len([]rune(e.Answer)) > 256 {
			e.Answer = string([]rune(e.Answer)[:256]) + "…"
		}
		if rule := p.QualityRule; rule.MaxTTFT > 0 || rule.MinTPS > 0 || rule.MaxErrorPercent > 0 {
			e.QualityMetrics, err = s.planRepo.QualityMetrics(ctx, p.AccountID, model)
			if err != nil {
				return err
			}
		}
		p.QualityRule.judge(&e, sample.Status == "success")
		p.QualityState[model] = p.QualityRule.advance(p.QualityState[model], e.Verdict)
		result.Evidence = append(result.Evidence, e)
		if e.Verdict == "unknown" && result.Status == "success" {
			result.Status = "unknown"
		}
		if e.Verdict == "failed" {
			result.Status = "failed"
		}
	}
	result.FinishedAt = time.Now()
	result.LatencyMs = result.FinishedAt.Sub(start).Milliseconds()
	next, err := computeNextRun(p.CronExpression, result.FinishedAt)
	if err != nil {
		return err
	}
	return s.planRepo.FinishQuality(ctx, p, result, next)
}
