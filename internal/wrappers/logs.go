package wrappers

import (
	"io"

	"github.com/checkmarxDev/logs/pkg/web/helpers"
)

type LogsWrapper interface {
	GetURL() (io.ReadCloser, *helpers.WebError, error)
}
