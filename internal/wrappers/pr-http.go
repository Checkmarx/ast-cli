package wrappers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParsePRDecorationResponse = "Failed to parse PR Decoration response."
)

type PRHTTPWrapper struct {
	path string
}

func NewHTTPPRWrapper(path string) PRWrapper {
	return &PRHTTPWrapper{
		path: path,
	}
}

func (r *PRHTTPWrapper) PostPRDecoration(model *PRModel) (
	*PRResponseModel,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		log.Println("Failed to marshal request")
	}
	log.Printf("Sending PR decoration request for scanID: %s\n", model.ScanID)
	resp, err := SendHTTPRequestWithJsonContentType(http.MethodPost, r.path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	return handlePRResponseWithBody(resp, err)
}

func handlePRResponseWithBody(resp *http.Response, err error) (*PRResponseModel, *WebError, error) {
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParsePRDecorationResponse)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := PRResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParsePRDecorationResponse)
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("PR Decoratrion POST not found")
	default:
		return nil, nil, errors.Errorf("Response status code %d", resp.StatusCode)
	}
}
