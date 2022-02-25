package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type CodeBashingMockWrapper struct{}

func (r CodeBashingMockWrapper) GetCodeBashingLinks(params map[string]string, codeBashingUrl *string) (*[]wrappers.CodeBashingCollection, *wrappers.WebError, error) {
	collection := &wrappers.CodeBashingCollection{
		Path:        "http://example.com/courses/php/lessons/dom_xss",
		CweID:       "CWE-79",
		Language:    "PHP",
		CxQueryName: "Reflected_XSS_All_Clients",
	}
	ret := []wrappers.CodeBashingCollection{*collection}
	return &ret, nil, nil
}

func (r CodeBashingMockWrapper) GetCodeBashingURL(field string) (
	*string, error,
) {
	field = "MOCK"
	return &field, nil
}

func (r CodeBashingMockWrapper) BuildCodeBashingParams([]wrappers.CodeBashingParamsCollection) (map[string]string, error){
	return nil,nil
}
