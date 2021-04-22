package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/checkmarxDev/logs/pkg/web/helpers"
	"github.com/checkmarxDev/logs/pkg/web/path/logs"
	"github.com/pkg/errors"
)

type LogsHTTPWrapper struct {
	basePath             string
	engineLogsPathFormat string
}

const (
	DownloadLogsTimeoutSeconds      = 60
	failedToParseDownloadLogsResult = "failed to get url"
	failedToParseDownloadResult     = "failed to parse download engine log result"
)

func NewLogsWrapper(basePath, logPathFormat string) LogsWrapper {
	return &LogsHTTPWrapper{basePath: basePath,
		engineLogsPathFormat: logPathFormat,
	}
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
			true, DownloadLogsTimeoutSeconds)
		if err != nil {
			return nil, nil, err
		}

		if downloadResp.StatusCode != http.StatusOK {
			defer downloadResp.Body.Close()
			return nil, nil, errors.Errorf("downloading logs after retrieving url got response status code %d",
				downloadResp.StatusCode)
		}

		return downloadResp.Body, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (l *LogsHTTPWrapper) DownloadEngineLog(scanID, engine string) (io.ReadCloser, *helpers.WebError, error) {
	path := fmt.Sprintf(l.engineLogsPathFormat, scanID, engine)
	resp, err := SendHTTPRequest(http.MethodGet, path, nil, true, DefaultTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}

	switch resp.StatusCode {
	case http.StatusInternalServerError:
		return nil, nil, errors.New("internal server error")
	case http.StatusNotFound, http.StatusBadRequest:
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		errorModel := &helpers.WebError{}
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
