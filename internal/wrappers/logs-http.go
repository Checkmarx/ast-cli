package wrappers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/checkmarxDev/logs/pkg/web/helpers"
	"github.com/pkg/errors"
)

type LogsHTTPWrapper struct {
	path string
}

const (
	failedToDecodeLogErr = "failed decoding log error"
	failedToReadLog      = "failed to read log"
	failedDownloadingLog = "failed to download log"
)

func NewLogsWrapper(path string) LogsWrapper {
	return &LogsHTTPWrapper{path: path}
}

func (l *LogsHTTPWrapper) GetLog(scanId, scanType string) (string, error) {
	reportPath := fmt.Sprintf("%s/%s/%s", l.path, scanId, scanType)
	resp, err := SendHTTPRequest(http.MethodGet, reportPath, nil, true, DefaultTimeoutSeconds)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := &helpers.WebError{}
		err = decoder.Decode(errorModel)
		if err != nil {
			return "", errors.Wrapf(err, failedToDecodeLogErr)
		}
		return "", errors.Errorf("%s: CODE: %d, %s", failedDownloadingLog, errorModel.Code, errorModel.Message)
	case http.StatusOK:
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", errors.Wrapf(err, failedToReadLog)
		}
		return string(bodyBytes), nil
	default:
		return "", errors.Errorf("response status code %d", resp.StatusCode)
	}
}
