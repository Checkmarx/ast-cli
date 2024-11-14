package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type PRMockWrapper struct {
}

func (pr *PRMockWrapper) PostPRDecoration(model *wrappers.PRModel) (
	string,
	*wrappers.WebError,
	error,
) {
	return "PR comment created successfully.", nil, nil
}

func (pr *PRMockWrapper) PostGitlabPRDecoration(model *wrappers.GitlabPRModel) (string, *wrappers.WebError, error) {
	return "MR comment created successfully.", nil, nil
}

func (pr *PRMockWrapper) PostAzurePRDecoration(model *wrappers.AzurePRModel) (string, *wrappers.WebError, error) {
	return "PR comment created successfully.", nil, nil
}
