package wrappers

import (
	"io"

	"github.com/checkmarxDev/sast-scan-inc/pkg/api/v1/rest"
)

type SastMetadataWrapper interface {
	DownloadEngineLog(scanID string) (io.ReadCloser, *rest.Error, error)
}
