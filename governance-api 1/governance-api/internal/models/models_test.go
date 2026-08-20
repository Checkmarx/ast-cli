package models

import (
	"testing"
)

func TestSeverityFromAction(t *testing.T) {
	cases := []struct {
		action string
		want   string
	}{
		{"BLOCK", "high"},
		{"ALERT", "medium"},
		{"ALLOW", "low"},
		{"",      "low"},
	}
	for _, tc := range cases {
		got := SeverityFromAction(tc.action)
		if got != tc.want {
			t.Errorf("SeverityFromAction(%q) = %q, want %q", tc.action, got, tc.want)
		}
	}
}

func TestGovernancePack_JSON(t *testing.T) {
	pack := GovernancePack{
		PackVersion: 5,
		Policies: []PackPolicy{
			{
				PolicyID:    "pol_001",
				PolicyName:  "Block .env",
				PolicyVersion: 2,
				Type:        "filesystem",
				Action:      "BLOCK",
				MatchTarget: "path",
				Pattern:     `\.env$`,
				AgentMessage: "Use secrets manager",
			},
		},
	}
	if len(pack.Policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(pack.Policies))
	}
	if pack.Policies[0].PolicyID != "pol_001" {
		t.Errorf("unexpected policyId: %s", pack.Policies[0].PolicyID)
	}
}
