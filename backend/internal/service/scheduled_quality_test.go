package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScheduledQualityHysteresis(t *testing.T) {
	rule := QualityRule{Models: []string{"m"}, Action: "model", Failures: 2, Passes: 2, MaxTTFT: 200, MinTPS: 4, MaxErrorPercent: 25}
	require.NoError(t, rule.validate())
	state := QualityState{}
	for _, v := range []string{"failed", "unknown", "failed"} {
		state = rule.advance(state, v)
	}
	require.False(t, state.Held)
	state = rule.advance(state, "failed")
	require.True(t, state.Held)
	state = rule.advance(state, "passed")
	require.True(t, state.Held)
	state = rule.advance(state, "passed")
	require.False(t, state.Held)
	for _, tc := range []struct {
		answer          string
		ok              bool
		metrics         QualityMetrics
		verdict, reason string
	}{
		{"12", true, QualityMetrics{}, "passed", ""},
		{"13", true, QualityMetrics{}, "failed", "answer_mismatch"},
		{"12", false, QualityMetrics{}, "unknown", ""},
		{"12", true, QualityMetrics{TTFT: qualityFloat(201)}, "failed", "ttft"},
		{"12", true, QualityMetrics{TPS: qualityFloat(3)}, "failed", "tps"},
		{"12", true, QualityMetrics{ErrorPercent: qualityFloat(26)}, "failed", "upstream_errors"},
	} {
		e := QualityEvidence{Expected: "12", Answer: tc.answer, QualityMetrics: tc.metrics}
		rule.judge(&e, tc.ok)
		require.Equal(t, tc.verdict, e.Verdict)
		require.Equal(t, tc.reason, e.Reason)
	}
	rule.Action = "observe"
	require.False(t, rule.advance(QualityState{Failures: 2}, "failed").Held)
}

func qualityFloat(v float64) *float64 { return &v }

func TestScheduledQualityRouting(t *testing.T) {
	group := int64(7)
	account := &Account{Status: StatusActive, Schedulable: true, QualityHolds: map[string]bool{"model:m": true, "group:7": true}}
	require.False(t, account.IsSchedulableForModel("m"))
	require.True(t, account.IsSchedulableForModel("other"))
	require.False(t, account.IsSchedulableForModelWithContext(withQualityGroup(context.Background(), &group), "other"))
	account.QualityHolds = map[string]bool{"account": true}
	require.False(t, account.IsSchedulable())
	account.Schedulable = false
	account.QualityHolds = nil
	require.False(t, account.IsSchedulable())
}
