package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	importsPath       = "/imports"
	successfulMessage = "The SARIF results were successfully imported into project %s importID: %s"
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
		return "", getError(decoder, errorConstants.StatusForbidden)
	case http.StatusUnauthorized:
		return "", getError(decoder, errorConstants.StatusUnauthorized)
	case http.StatusInternalServerError:
		byorErrorModel := ByorErrorModel{}
		decodeErr := decoder.Decode(&byorErrorModel)
		if decodeErr != nil {
			return "", errors.Errorf(fmt.Sprintf(errorConstants.ImportSarifFileErrorMessageWithMessage, http.StatusInternalServerError, "Error decoding byor error model"))
		}
		return "", errors.Errorf(fmt.Sprintf(errorConstants.ImportSarifFileErrorMessageWithMessage, byorErrorModel.Code, byorErrorModel.Message))
	case http.StatusOK:
		model := CreateImportsResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return model.ImportID, errors.Errorf(errorConstants.ImportSarifFileError)
		}
		logger.Printf(successfulMessage, projectID, model.ImportID)
		return model.ImportID, nil
	default:
		return "", errors.Errorf(fmt.Sprintf(errorConstants.ImportSarifFileErrorMessageWithMessage, resp.StatusCode, ""))
	}
}

func getError(decoder *json.Decoder, errorMessage string) error {
	errorModel := ByorErrorModel{}
	err := decoder.Decode(&errorModel)
	if err != nil {
		return errors.Errorf("Parsing error model failed - %s", err.Error())
	}
	logger.PrintIfVerbose(errorModel.Message)
	return errors.Errorf(errorMessage)
}

type ByorErrorModel struct {
	Message string   `json:"message"`
	Code    int      `json:"code"`
	Details []string `json:"details"`
}
