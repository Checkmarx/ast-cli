package wrappers

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/checkmarxDev/logs/pkg/web/helpers"
)

type LogsMockWrapper struct{}

func (LogsMockWrapper) DownloadEngineLog(string, string) (io.ReadCloser, *helpers.WebError, error) {
	return ioutil.NopCloser(strings.NewReader(MockContent)), nil, nil
}

func (LogsMockWrapper) GetURL() (io.ReadCloser, *helpers.WebError, error) {
	return ioutil.NopCloser(strings.NewReader(MockContent)), nil, nil
}
