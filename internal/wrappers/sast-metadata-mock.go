package wrappers

import (
	"github.com/checkmarxDev/sast-metadata/pkg/api/v1/rest"
)

type SastMetadataMockWrapper struct{}

const MockContent = "mock"

func (SastMetadataMockWrapper) GetScanInfo(string) (*rest.ScanInfo, *rest.Error, error) {
	return &rest.ScanInfo{
		ScanID:        "123",
		IsIncremental: false,
	}, nil, nil
}

func (SastMetadataMockWrapper) GetMetrics(scanID string) (*rest.Metrics, *rest.Error, error) {
	return &rest.Metrics{
		ScanID:     scanID,
		MemoryPeak: 123,
	}, nil, nil
}
