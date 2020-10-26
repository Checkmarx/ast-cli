package wrappers

import (
	"time"

	"github.com/google/uuid"

	"github.com/checkmarxDev/sast-rm/pkg/api/rest"
)

type SastRmMockWrapper struct {
}

func (s *SastRmMockWrapper) GetPools() ([]*rest.Pool, error) {
	return []*rest.Pool{
		{ID: "id", Description: "Description", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}, nil
}

func (s *SastRmMockWrapper) GetPoolEngines(id string) ([]string, error) {
	return []string{
		uuid.New().String(),
		uuid.New().String(),
	}, nil
}

func (s *SastRmMockWrapper) GetPoolProjects(id string) ([]string, error) {
	return []string{
		uuid.New().String(),
		uuid.New().String(),
	}, nil
}

func (s *SastRmMockWrapper) GetPoolEngineTags(id string) (map[string]string, error) {
	return map[string]string{
		uuid.New().String(): uuid.New().String(),
	}, nil
}

func (s *SastRmMockWrapper) GetPoolProjectTags(id string) (map[string]string, error) {
	return map[string]string{
		uuid.New().String(): uuid.New().String(),
	}, nil
}

func (s *SastRmMockWrapper) SetPoolEngines(id string, value []string) ([]string, error) {
	return value, nil
}

func (s *SastRmMockWrapper) SetPoolProjects(id string, value []string) ([]string, error) {
	return value, nil
}

func (s *SastRmMockWrapper) SetPoolEngineTags(id string, value map[string]string) (map[string]string, error) {
	return value, nil
}

func (s *SastRmMockWrapper) SetPoolProjectTags(id string, value map[string]string) (map[string]string, error) {
	return value, nil
}

func (s *SastRmMockWrapper) GetStats(_ StatResolution) ([]*rest.Metric, error) {
	return []*rest.Metric{
		{
			ScansPending:   1,
			ScansRunning:   1,
			ScansOrphan:    1,
			ScansTotal:     1,
			EnginesWaiting: 1,
			EnginesRunning: 1,
			EnginesTotal:   1,
			Time:           time.Now(),
		},
		{
			ScansPending:   2,
			ScansRunning:   2,
			ScansOrphan:    2,
			ScansTotal:     2,
			EnginesWaiting: 2,
			EnginesRunning: 2,
			EnginesTotal:   2,
			Time:           time.Now().Add(-time.Minute),
		},
		{
			ScansPending:   3,
			ScansRunning:   3,
			ScansOrphan:    3,
			ScansTotal:     3,
			EnginesWaiting: 3,
			EnginesRunning: 3,
			EnginesTotal:   3,
			Time:           time.Now().Add(-time.Minute * 2), //nolint:gomnd
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
			Engine:      "",
			Properties: map[string]string{
				"lala":   "topola",
				"trally": "valy",
			},
		},
		{
			ID:          "a0233599-44ce-44a3-b73f-b83d84c6dda1",
			State:       "Queued",
			QueuedAt:    now,
			AllocatedAt: &now,
			Engine:      "59698599-e2ff-4efc-8fb1-35599c0ba7fa",
		},
	}, nil
}

func (s *SastRmMockWrapper) GetEngines() ([]*rest.Engine, error) {
	return []*rest.Engine{
		{
			ID:           "59698599-e2ff-4efc-8fb1-35599c0ba7fa",
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
			ID:           "33698511-e2ff-4efc-8fb1-35599c0ba755",
			Status:       "Ready",
			RegisteredAt: time.Now(),
			UpdatedAt:    time.Now(),
		},
	}, nil
}
