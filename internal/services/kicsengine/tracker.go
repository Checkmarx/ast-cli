package kicsengine

import (
	"sync"

	"github.com/Checkmarx/kics/v2/pkg/model"
)

// tracker satisfies both kics.Tracker and engine.Tracker. KICS's own implementation lives in
// its internal/ tree and cannot be imported, so the CLI supplies its own.
//
// The counters are not cosmetic: they populate the counter fields of the JSON report, which
// the container engine also emits, so they have to be maintained faithfully.
type tracker struct {
	mu sync.Mutex

	foundFiles       int
	foundCountLines  int
	parsedFiles      int
	parsedCountLines int
	ignoreCountLines int

	loadedQueries    int
	executingQueries int
	executedQueries  int

	failedSimilarityID int

	outputLines int
}

func newTracker(outputLines int) *tracker {
	return &tracker{outputLines: outputLines}
}

func (t *tracker) TrackFileFound(string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.foundFiles++
}

func (t *tracker) TrackFileParse(string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.parsedFiles++
}

func (t *tracker) TrackFileFoundCountLines(countLines int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.foundCountLines += countLines
}

func (t *tracker) TrackFileParseCountLines(countLines int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.parsedCountLines += countLines
}

func (t *tracker) TrackFileIgnoreCountLines(countLines int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ignoreCountLines += countLines
}

func (t *tracker) TrackQueryLoad(queryAggregation int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.loadedQueries += queryAggregation
}

func (t *tracker) TrackQueryExecuting(queryAggregation int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.executingQueries += queryAggregation
}

func (t *tracker) TrackQueryExecution(queryAggregation int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.executedQueries += queryAggregation
}

func (t *tracker) FailedComputeSimilarityID() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failedSimilarityID++
}

// TrackScanPath, TrackScanSecret, FailedDetectLine and FailedComputeOldSimilarityID feed
// KICS's telemetry rather than the report, so they are intentionally no-ops here.
func (t *tracker) TrackScanPath()                {}
func (t *tracker) TrackScanSecret()              {}
func (t *tracker) FailedDetectLine()             {}
func (t *tracker) FailedComputeOldSimilarityID() {}

func (t *tracker) GetOutputLines() int { return t.outputLines }

// counters snapshots the tracked values in the shape model.CreateSummary expects.
func (t *tracker) counters() model.Counters {
	t.mu.Lock()
	defer t.mu.Unlock()
	return model.Counters{
		ScannedFiles:           t.foundFiles,
		ScannedFilesLines:      t.foundCountLines,
		ParsedFiles:            t.parsedFiles,
		ParsedFilesLines:       t.parsedCountLines,
		IgnoredFilesLines:      t.ignoreCountLines,
		TotalQueries:           t.loadedQueries,
		FailedToExecuteQueries: t.executingQueries - t.executedQueries,
		FailedSimilarityID:     t.failedSimilarityID,
	}
}
