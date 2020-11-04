package wrappers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/checkmarxDev/logs/pkg/web/helpers"
	"github.com/checkmarxDev/logs/pkg/web/path/logs"
	"github.com/pkg/errors"
)

type LogsHTTPWrapper struct {
	basePath string
}

const (
	DownloadLogsTimeoutSeconds      = 60
	failedToParseDownloadLogsResult = "failed to get url"
)

func NewLogsWrapper(basePath string) LogsWrapper {
	return &LogsHTTPWrapper{basePath: basePath}
}

func (l *LogsHTTPWrapper) GetURL() (io.ReadCloser, *helpers.WebError, error) {
	resp, err := SendHTTPRequest(http.MethodGet, l.basePath, nil, true, DownloadLogsTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := &helpers.WebError{}
		err = decoder.Decode(errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseDownloadLogsResult)
		}

		return nil, errorModel, nil
	case http.StatusOK:
		model := &logs.GetLogsResponse{}
		err := decoder.Decode(model)
		if err != nil {
			return nil, nil, errors.Wrap(err, failedToParseDownloadLogsResult)
		}

		downloadResp, err := SendHTTPRequestByFullURL(http.MethodGet, model.URL, nil,
			false, DefaultTimeoutSeconds)
		if err != nil {
			return nil, nil, err
		}

		if downloadResp.StatusCode != http.StatusOK {
			defer resp.Body.Close()
			return nil, nil, errors.Errorf("response status code %d", downloadResp.StatusCode)
		}

		return downloadResp.Body, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
