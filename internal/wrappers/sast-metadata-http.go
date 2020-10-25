package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/checkmarxDev/sast-scan-inc/pkg/api/v1/rest"
	"github.com/pkg/errors"
)

type SastMetadataHTTPWrapper struct {
	pathFormat string
}

const failedToParseDownloadResult = "failed to parse download engine log result"

func NewSastMetadataHTTPWrapper(pathFormat string) SastMetadataWrapper {
	return &SastMetadataHTTPWrapper{
		pathFormat: pathFormat,
	}
}

func (s *SastMetadataHTTPWrapper) DownloadEngineLog(scanID string) (io.ReadCloser, *rest.Error, error) {
	resp, err := SendHTTPRequest(http.MethodGet, fmt.Sprintf(s.pathFormat, scanID), nil, true, DefaultTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}

	switch resp.StatusCode {
	case http.StatusNotFound, http.StatusInternalServerError:
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		errorModel := &rest.Error{}
		err = decoder.Decode(errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseDownloadResult)
		}

		return nil, errorModel, nil
	case http.StatusOK:
		return resp.Body, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
