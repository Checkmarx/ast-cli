package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type RisksOverviewHTTPWrapper struct {
	path string
}

func NewHTTPRisksOverviewWrapper(path string) RisksOverviewWrapper {
	return &RisksOverviewHTTPWrapper{
		path: path,
	}
}

func (r *RisksOverviewHTTPWrapper) GetAllAPISecRisksByScanID(scanID string) (
	*APISecResult,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := fmt.Sprintf(r.path, scanID)
	resp, err := SendHTTPRequest(http.MethodGet, path, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := GetWebError(decoder)
		return nil, &errorModel, nil
	case http.StatusOK:
		model := APISecResult{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
