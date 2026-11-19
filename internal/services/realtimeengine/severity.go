package realtimeengine

import (
	"fmt"
	"strings"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
)

// AcceptedSeverities lists the values the --severity-filter flag accepts, in display order.
var AcceptedSeverities = []string{"critical", "high", "medium", "low", "info"}

// SeverityFilter is the set of lowercase severities a realtime command should return.
// A nil filter means "no filtering": every severity is allowed.
type SeverityFilter map[string]struct{}

// ParseSeverityFilter parses the raw --severity-filter flag value.
// An empty or whitespace-only value returns a nil filter (allow all).
// Tokens are trimmed, lowercased and blank tokens are skipped, matching the
// other comma-separated flags on `cx scan`. Every unknown token is collected
// into a single RealtimeEngineError that also lists the accepted values.
func ParseSeverityFilter(raw string) (SeverityFilter, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	accepted := make(map[string]struct{}, len(AcceptedSeverities))
	for _, s := range AcceptedSeverities {
		accepted[s] = struct{}{}
	}

	filter := make(SeverityFilter)
	var invalid []string
	for _, token := range strings.Split(raw, ",") {
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" {
			continue
		}
		if _, ok := accepted[token]; !ok {
			invalid = append(invalid, token)
			continue
		}
		filter[token] = struct{}{}
	}

	if len(invalid) > 0 {
		return nil, errorconstants.NewRealtimeEngineError(fmt.Sprintf(
			"invalid severity-filter value(s): %s. Accepted values: %s",
			strings.Join(invalid, ", "), strings.Join(AcceptedSeverities, ", "),
		)).Error()
	}

	if len(filter) == 0 {
		return nil, nil
	}
	return filter, nil
}

// Allows reports whether a finding with the given severity should be returned.
// A nil filter allows everything; comparison is case insensitive.
func (f SeverityFilter) Allows(severity string) bool {
	if f == nil {
		return true
	}
	_, ok := f[strings.ToLower(strings.TrimSpace(severity))]
	return ok
}
