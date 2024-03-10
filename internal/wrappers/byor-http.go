package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	importsPath = "/imports"

	successfulMessage = "The SARIF results were successfully imported into project %s"
)

type ByorHTTPWrapper struct {
	path          string
	clientTimeout uint
}

func NewByorHTTPWrapper(path string) ByorWrapper {
	return &ByorHTTPWrapper{
		path:          path,
		clientTimeout: viper.GetUint(commonParams.ClientTimeoutKey),
	}
}
func (b *ByorHTTPWrapper) Import(projectID, uploadURL string) (string, error) {
	req := CreateImportsRequest{
		ProjectID: projectID,
		UploadURL: uploadURL,
	}
	jsonBytes, _ := json.Marshal(req)
	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, b.path+importsPath, bytes.NewBuffer(jsonBytes), true, b.clientTimeout)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	decoder := json.NewDecoder(resp.Body)
	switch resp.StatusCode {
	case http.StatusForbidden:
		return "", getError(decoder, "You are not allowed to make this request")
	case http.StatusUnauthorized:
		return "", getError(decoder, "You are not authorized to make this request")
	case http.StatusInternalServerError:
		return "", getError(decoder, "An error occurred during this request")
	case http.StatusOK:
		model := CreateImportsResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return "", errors.Errorf("Parsing upload model failed - %s", err.Error())
		}
		logger.Printf(successfulMessage, projectID)
		return model.ImportID, nil
	default:
		return "", errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func getError(decoder *json.Decoder, errorMessage string) error {
	errorModel := ErrorModel{}
	err := decoder.Decode(&errorModel)
	if err != nil {
		return errors.Errorf("Parsing error model failed - %s", err.Error())
	}
	logger.PrintIfVerbose(errorModel.Message)
	return errors.Errorf(errorMessage)
}
