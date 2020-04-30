package wrappers

import (
	"time"

	"github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"
)

type SastRmMockWrapper struct {
}

func (s *SastRmMockWrapper) GetStats(metric StatMetric, resolution StatResolution) ([]*rest.Counter, error) {
	return []*rest.Counter{
		{
			Time:  time.Now(),
			Count: 1,
		},
	}, nil
}

func (s *SastRmMockWrapper) GetScans() ([]*rest.Scan, error) {
	return []*rest.Scan{
		{
			ID:       "kuku",
			State:    "Queued",
			QueuedAt: time.Now(),
		},
	}, nil
}

func (s *SastRmMockWrapper) GetEngines() ([]*rest.Engine, error) {
	return []*rest.Engine{
		{
			ID:           "riku",
			Status:       "Ready",
			RegisteredAt: time.Now(),
			UpdatedAt:    time.Now(),
		},
	}, nil
}
