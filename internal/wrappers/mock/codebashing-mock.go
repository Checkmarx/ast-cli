package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type CodeBashingMockWrapper struct{}

func (r CodeBashingMockWrapper) GetCodeBashingLinks(_ map[string]string) (
	*[]wrappers.CodeBashingCollection,
	*wrappers.WebError,
	error,
) {
	const mock = "MOCK"
	collection := &wrappers.CodeBashingCollection{
		Path:        mock,
		CweId:       mock,
		Lang:        mock,
		CxQueryName: mock,
	}
	ret := []wrappers.CodeBashingCollection{*collection}
	return &ret, nil, nil
}

func (r ResultsMockWrapper) GetCodeBashingLinks(_ map[string]string) (
	*[]wrappers.CodeBashingCollection,
	*wrappers.WebError,
	error,
) {
	collection := &wrappers.CodeBashingCollection{
		Path:        "http://example.com/courses/php/lessons/dom_xss",
		CweId:       "CWE-79",
		Lang:        "PHP",
		CxQueryName: "Reflected_XSS_All_Clients",
	}
	ret := []wrappers.CodeBashingCollection{*collection}
	return &ret, nil, nil
}
