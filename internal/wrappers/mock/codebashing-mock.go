package mock

import (
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type CodeBashingMockWrapper struct{}

func (r CodeBashingMockWrapper) GetCodeBashingLinks(queryID, codeBashingURL string) (*[]wrappers.CodeBashingCollection, *wrappers.WebError, error) {
	if queryID == "" {
		return nil, &wrappers.WebError{Message: "Cannot GET /lessons/mapping/"}, nil
	}
	codeBashingURL = commonParams.CodeBashingPathEnv

	if queryID == "11666704984804998184" {
		collection := &wrappers.CodeBashingCollection{
			Path: "/app/home",
		}
		ret := []wrappers.CodeBashingCollection{*collection}
		return &ret, nil, nil
	}

	collection := &wrappers.CodeBashingCollection{
		Path: "http://example.com/courses/php/lessons/dom_xss",
	}
	ret := []wrappers.CodeBashingCollection{*collection}
	return &ret, nil, nil
}

func (r CodeBashingMockWrapper) GetCodeBashingURL(field string) (
	string, error,
) {
	return "MOCK", nil
}

func (r CodeBashingMockWrapper) BuildCodeBashingParams(wrappers.CodeBashingParamsCollection) (map[string]string, error) {
	return nil, nil
}
