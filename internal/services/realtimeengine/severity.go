package realtimeengine

import (
	"fmt"
	"slices"
	"strings"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/cobra"
)

var severityOrder = []string{"info", "low", "medium", "high", "critical"}

// SeverityThreshold is the rank of the lowest severity to return (1 = info ... 5 = critical); zero means no filtering.
type SeverityThreshold int

// ParseSeverityThreshold parses a single severity level, ignoring case; empty input returns zero (no filtering).
func ParseSeverityThreshold(raw string) (SeverityThreshold, error) {
	level := strings.ToLower(strings.TrimSpace(raw))
	if level == "" {
		return 0, nil
	}
	rank := slices.Index(severityOrder, level) + 1
	if rank == 0 {
		return 0, errorconstants.NewRealtimeEngineError(fmt.Sprintf(
			"invalid severity-threshold value %q. Use a single level: critical, high, medium, low or info", strings.TrimSpace(raw),
		)).Error()
	}
	return SeverityThreshold(rank), nil
}

// SeverityThresholdFromCmd parses the --severity-threshold flag of cmd.
func SeverityThresholdFromCmd(cmd *cobra.Command) (SeverityThreshold, error) {
	raw, _ := cmd.Flags().GetString(commonParams.SeverityThresholdFlag)
	return ParseSeverityThreshold(raw)
}

// Allows reports whether a severity is at or above the threshold, ignoring case; unrecognised severities are kept.
func (t SeverityThreshold) Allows(severity string) bool {
	rank := slices.Index(severityOrder, strings.ToLower(strings.TrimSpace(severity))) + 1
	return t == 0 || rank == 0 || SeverityThreshold(rank) >= t
}

// FilterBySeverity returns the items at or above t; a zero threshold or empty input is returned unchanged.
func FilterBySeverity[T any](items []T, t SeverityThreshold, severity func(T) string) []T {
	if t == 0 || len(items) == 0 {
		return items
	}
	kept := make([]T, 0, len(items))
	for _, item := range items {
		if t.Allows(severity(item)) {
			kept = append(kept, item)
		}
	}
	return kept
}
