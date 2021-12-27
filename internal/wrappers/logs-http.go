package wrappers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
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

func (l *LogsHTTPWrapper) GetLog(scanID, scanType string) (string, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	reportPath := fmt.Sprintf("%s/%s/%s", l.path, scanID, scanType)
	resp, err := SendHTTPRequest(http.MethodGet, reportPath, nil, true, clientTimeout)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := &WebError{}
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
