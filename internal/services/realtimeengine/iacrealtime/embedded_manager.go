package iacrealtime

import (
	"context"

	"github.com/checkmarx/ast-cli/internal/services/kicsengine"
)

// RunEmbeddedScan scans with the KICS engine compiled into the CLI. It needs no container
// runtime and no downloaded binary, so unlike the container backend it cannot fail for
// environmental reasons - any error here is a real scan failure.
//
// Passing the source file lets the engine narrow both the query set and the parser set to what
// can actually match it, which is the difference between a couple of seconds and tens of
// seconds. Ambiguous extensions fall back to loading everything, exactly like the container.
//
// The engine writes the same results.json the container produced into tempDir, so the result
// handling below this point is shared with the container backend.
func (s *Scanner) RunEmbeddedScan(ctx context.Context, tempDir, filePath string) ([]IacRealtimeResult, error) {
	if err := kicsengine.Scan(ctx, kicsengine.Options{
		ScanPath:   tempDir,
		OutputDir:  tempDir,
		SourceFile: filePath,
	}); err != nil {
		return nil, err
	}
	return s.processResults(tempDir, filePath)
}
