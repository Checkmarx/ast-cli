package realtimeengine

import (
	"fmt"
	"slices"
	"strings"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/cobra"
)

var acceptedSeverities = []string{"critical", "high", "medium", "low", "info"}

// SeverityFilter is the set of lowercase severities to return; nil means no filtering.
type SeverityFilter map[string]struct{}

// ParseSeverityFilter parses a comma-separated severity list; empty or separator-only input returns a nil filter.
func ParseSeverityFilter(raw string) (SeverityFilter, error) {
	filter := SeverityFilter{}
	var invalid []string
	for _, token := range strings.Split(raw, ",") {
		token = strings.ToLower(strings.TrimSpace(token))
		switch {
		case token == "":
		case slices.Contains(acceptedSeverities, token):
			filter[token] = struct{}{}
		default:
			invalid = append(invalid, token)
		}
	}

	if len(invalid) > 0 {
		return nil, errorconstants.NewRealtimeEngineError(fmt.Sprintf(
			"invalid severity-filter value(s): %s. Accepted values: %s",
			strings.Join(invalid, ", "), strings.Join(acceptedSeverities, ", "),
		)).Error()
	}
	if len(filter) == 0 {
		return nil, nil
	}
	return filter, nil
}

// SeverityFilterFromCmd parses the --severity-filter flag of cmd.
func SeverityFilterFromCmd(cmd *cobra.Command) (SeverityFilter, error) {
	raw, _ := cmd.Flags().GetString(commonParams.SeverityFilterFlag)
	return ParseSeverityFilter(raw)
}

// Allows reports whether a finding of the given severity should be returned, ignoring case.
func (f SeverityFilter) Allows(severity string) bool {
	if f == nil {
		return true
	}
	_, ok := f[strings.ToLower(strings.TrimSpace(severity))]
	return ok
}

// FilterBySeverity returns the items allowed by f; a nil filter or empty input is returned unchanged.
func FilterBySeverity[T any](items []T, f SeverityFilter, severity func(T) string) []T {
	if f == nil || len(items) == 0 {
		return items
	}
	kept := make([]T, 0, len(items))
	for _, item := range items {
		if f.Allows(severity(item)) {
			kept = append(kept, item)
		}
	}
	return kept
}
