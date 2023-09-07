package wrappers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParseBflResponse = "Failed to parse BFL response."
)

type BflHTTPWrapper struct {
	path string
}

func NewBflHTTPWrapper(path string) BflWrapper {
	return &BflHTTPWrapper{
		path: path,
	}
}

func (r *BflHTTPWrapper) GetBflByScanIDAndQueryID(params map[string]string) (
	*BFLResponseModel,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	log.Println(fmt.Sprintf("Fetching the best fix location for QueryID: %s", params[commonParams.QueryIDQueryParam]))
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	return handleBflResponseWithBody(resp, err)
}

func handleBflResponseWithBody(resp *http.Response, err error) (*BFLResponseModel, *WebError, error) {
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseBflResponse)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := BFLResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseBflResponse)
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("Best Fix Location not found.")
	default:
		return nil, nil, errors.Errorf("Response status code %d", resp.StatusCode)
	}
}
