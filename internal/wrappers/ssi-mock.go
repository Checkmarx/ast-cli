package wrappers

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/checkmarxDev/sast-scan-inc/pkg/api/v1/rest"
)

type SSIMockWrapper struct{}

const MockContent = "mock"

func (SSIMockWrapper) DownloadEngineLog(string) (io.ReadCloser, *rest.Error, error) {
	return ioutil.NopCloser(strings.NewReader(MockContent)), nil, nil
}
