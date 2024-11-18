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
	githubPath          string
	gitlabPath          string
	azurePath           string
	bitbucketCloudPath  string
	bitbucketServerPath string
}

func NewHTTPPRWrapper(githubPath, gitlabPath, bitbucketCloudPath, bitbucketServerPath, azurePath string) PRWrapper {
	return &PRHTTPWrapper{
		githubPath:          githubPath,
		gitlabPath:          gitlabPath,
		azurePath:           azurePath,
		bitbucketCloudPath:  bitbucketCloudPath,
		bitbucketServerPath: bitbucketServerPath,
	}
}

func (r *PRHTTPWrapper) PostPRDecoration(model interface{}) (string, *WebError, error) {
	url, err := r.getPRDecorationURL(model)
	if err != nil {
		return "", nil, err
	}

	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return "", nil, err
	}
	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, url, bytes.NewBuffer(jsonBytes), true, clientTimeout)
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

func (r *PRHTTPWrapper) getPRDecorationURL(model interface{}) (string, error) {
	switch model.(type) {
	case *PRModel:
		return r.githubPath, nil
	case *GitlabPRModel:
		return r.gitlabPath, nil
	case *BitbucketCloudPRModel:
		return r.bitbucketCloudPath, nil
	case *BitbucketServerPRModel:
		return r.bitbucketServerPath, nil
	case *AzurePRModel:
		return r.azurePath, nil
	default:
		return "", errors.New("unsupported model type")
	}
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
