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
	now := time.Now()
	return []*rest.Scan{
		{
			ID:          "c0b64599-54da-44a3-b73f-b83d84c6dfe4",
			State:       "Queued",
			QueuedAt:    now,
			AllocatedAt: &now,
			Priority:    100,
			Engine:      "",
			Constrains: map[string]string{
				"lala":   "topola",
				"trally": "valy",
			},
		},
		{
			ID:          "a0233599-44ce-44a3-b73f-b83d84c6dda1",
			State:       "Queued",
			QueuedAt:    now,
			AllocatedAt: &now,
			Priority:    100,
			Engine:      "SAST.59698599-e2ff-4efc-8fb1-35599c0ba7fa",
		},
	}, nil
}

func (s *SastRmMockWrapper) GetEngines() ([]*rest.Engine, error) {
	return []*rest.Engine{
		{
			ID:           "SAST.59698599-e2ff-4efc-8fb1-35599c0ba7fa",
			Status:       "Running",
			ScanID:       "a0233599-44ce-44a3-b73f-b83d84c6dda1",
			RegisteredAt: time.Now(),
			UpdatedAt:    time.Now(),
			Properties: map[string]string{
				"lala":    "topola",
				"trally":  "valy",
				"name":    "some mane",
				"version": "2.0",
			},
		},
		{
			ID:           "SAST.33698511-e2ff-4efc-8fb1-35599c0ba755",
			Status:       "Ready",
			RegisteredAt: time.Now(),
			UpdatedAt:    time.Now(),
		},
	}, nil
}
