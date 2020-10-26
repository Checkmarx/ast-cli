package wrappers

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/checkmarxDev/sast-scan-inc/pkg/api/v1/rest"
)

type SastMetadataMockWrapper struct{}

const MockContent = "mock"

func (SastMetadataMockWrapper) DownloadEngineLog(string) (io.ReadCloser, *rest.Error, error) {
	return ioutil.NopCloser(strings.NewReader(MockContent)), nil, nil
}

func (SastMetadataMockWrapper) GetScanInfo(string) (*rest.ScanInfo, *rest.Error, error) {
	return &rest.ScanInfo{
		ScanID:        "123",
		IsIncremental: false,
	}, nil, nil
}
