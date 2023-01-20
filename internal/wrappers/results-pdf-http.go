package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
)

type PdfReportsPayload struct {
	ReportName string `json:"reportName" validate:"required"`
	ReportType string `json:"reportType" validate:"required"`
	FileFormat string `json:"fileFormat" validate:"required"`
	Data       struct {
		ScanId     string   `json:"scanId" validate:"required"`
		ProjectId  string   `json:"projectId" validate:"required"`
		BranchName string   `json:"branchName" validate:"required"`
		Host       string   `json:"host"`
		Sections   []string `json:"sections"`
		Scanners   []string `json:"scanners"`
		Email      []string `json:"email"`
	} `json:"data"`
}

type PdfReportsResponse struct {
	ReportId string `json:"reportId"`
}

type ResultsPdfReportsHttpWrapper struct {
	path string
}

func NewResultsPdfReportsHttpWrapper(path string) ResultsPdfReportsWrapper {
	return &ResultsPdfReportsHttpWrapper{
		path: path,
	}
}

func (r *ResultsPdfReportsHttpWrapper) GeneratePdfReport(payload PdfReportsPayload) (*PdfReportsResponse, *WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	params, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Failed to parse request body")
	}
	resp, err := SendHTTPRequest(http.MethodPost, r.path, bytes.NewBuffer(params), true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Error while requesting PDF report")
		}
		return nil, &errorModel, nil
	case http.StatusAccepted:
		model := PdfReportsResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Failed to parse response body")
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
