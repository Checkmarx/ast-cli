package wrappers

import (
	"io"

	"github.com/checkmarxDev/logs/pkg/web/helpers"
)

type LogsWrapper interface {
	DownloadEngineLog(scanID, engine string) (io.ReadCloser, *helpers.WebError, error)
	GetURL() (io.ReadCloser, *helpers.WebError, error)
}
