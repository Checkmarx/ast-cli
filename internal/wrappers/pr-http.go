package wrappers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParsePRDecorationResponse = "Failed to parse PR Decoration response."
)

type PRHTTPWrapper struct {
	githubPath string
	gitlabPath string
	azurePath  string
}

func NewHTTPPRWrapper(githubPath, gitlabPath, azurePath string) PRWrapper {
	return &PRHTTPWrapper{
		githubPath: githubPath,
		gitlabPath: gitlabPath,
		azurePath:  azurePath,
	}
}

func (r *PRHTTPWrapper) PostAzurePRDecoration(model *AzurePRModel) (
	string,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return "", nil, err
	}
	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, r.azurePath, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return handlePRResponseWithBody(resp, err)

}
func (r *PRHTTPWrapper) PostPRDecoration(model *PRModel) (
	string,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return "", nil, err
	}
	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, r.githubPath, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return handlePRResponseWithBody(resp, err)
}

func (r *PRHTTPWrapper) PostGitlabPRDecoration(model *GitlabPRModel) (
	string,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return "", nil, err
	}
	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, r.gitlabPath, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return handlePRResponseWithBody(resp, err)
}

func handlePRResponseWithBody(resp *http.Response, err error) (string, *WebError, error) {
	if err != nil {
		return "", nil, err
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
			return "", nil, errors.Wrapf(err, failedToParsePRDecorationResponse)
		}
		return "", &errorModel, nil
	case http.StatusCreated:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", nil, errors.Wrapf(err, failedToParsePRDecorationResponse)
		}
		return string(body), nil, nil
	case http.StatusNotFound:
		return "", nil, errors.Errorf("PR Decoratrion POST not found")
	default:
		return "", nil, errors.Errorf("Response status code %d", resp.StatusCode)
	}
}
