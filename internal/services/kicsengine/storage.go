package kicsengine

import (
	"context"
	"sync"

	"github.com/Checkmarx/kics/v2/pkg/model"
)

// memoryStorage satisfies kics.Storage. KICS's own memory storage is under internal/ and so
// is not importable; this is the same idea, scoped to a single scan.
type memoryStorage struct {
	mu              sync.Mutex
	vulnerabilities []model.Vulnerability
	files           model.FileMetadatas
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{}
}

func (s *memoryStorage) SaveFile(_ context.Context, metadata *model.FileMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = append(s.files, *metadata)
	return nil
}

func (s *memoryStorage) SaveVulnerabilities(_ context.Context, vulnerabilities []model.Vulnerability) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vulnerabilities = append(s.vulnerabilities, vulnerabilities...)
	return nil
}

func (s *memoryStorage) GetVulnerabilities(_ context.Context, _ string) ([]model.Vulnerability, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Vulnerability, len(s.vulnerabilities))
	copy(out, s.vulnerabilities)
	return out, nil
}

// GetScanSummary is part of kics.Storage but is only used by KICS's own reporting paths,
// which the CLI does not drive; the report is built from the vulnerabilities instead.
func (s *memoryStorage) GetScanSummary(_ context.Context, _ []string) ([]model.SeveritySummary, error) {
	return nil, nil
}
