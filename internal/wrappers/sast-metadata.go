package wrappers

import (
	"github.com/checkmarxDev/sast-metadata/pkg/api/v1/rest"
)

type SastMetadataWrapper interface {
	GetScanInfo(scanID string) (*rest.ScanInfo, *rest.Error, error)
	GetMetrics(scanID string) (*rest.Metrics, *rest.Error, error)
}
