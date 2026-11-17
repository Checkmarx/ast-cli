// Package telemetry provides OS-level ground-truth confirmation of whether a
// tool call actually took effect.
//
// The governance correlation sweeper used to infer third-party blocks purely
// from a missing PostToolUse marker after a timeout — a probabilistic guess that
// misfires whenever the marker is merely delayed. This package answers the same
// question from observable OS state instead:
//
//   - file tools  → did the target file actually change? (stat snapshot diff)
//   - shell tools → did a matching process actually start? (process table watch)
//
// Every function is detection-only: nothing here blocks, kills, or modifies a
// tool call. Results are advisory input to the sweeper's attribution logic.
package telemetry

import "time"

// Result is the outcome of a ground-truth confirmation attempt.
type Result int

const (
	// Inconclusive means telemetry could not observe the signal reliably —
	// callers must fall back to the existing hook-correlation heuristic.
	Inconclusive Result = iota
	// Confirmed means the action demonstrably happened (file changed / process ran),
	// so it was NOT blocked by anyone.
	Confirmed
	// NotObserved means telemetry watched for the full window and the action
	// demonstrably did not happen — a real block.
	NotObserved
)

func (r Result) String() string {
	switch r {
	case Confirmed:
		return "confirmed"
	case NotObserved:
		return "not_observed"
	default:
		return "inconclusive"
	}
}

// DetectionMethod names the provenance of a third-party enforcement verdict.
// It rides along on the ActivityEvent so the dashboard can distinguish
// ground-truth detections from timeout-based guesses.
const (
	MethodTelemetry       = "telemetry"
	MethodHookCorrelation = "hook_correlation"
)

// Method maps a Result to the DetectionMethod string reported to the backend.
func (r Result) Method() string {
	if r == Inconclusive {
		return MethodHookCorrelation
	}
	return MethodTelemetry
}

// DefaultWindow is how long a watcher observes before concluding NotObserved.
// Slightly longer than the correlation window so the sweeper never races it.
const DefaultWindow = 6 * time.Second

// pollInterval is how often watchers re-sample OS state within the window.
const pollInterval = 350 * time.Millisecond
