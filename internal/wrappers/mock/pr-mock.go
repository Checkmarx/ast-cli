package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type PRMockWrapper struct {
}

func (pr *PRMockWrapper) PostPRDecoration(model *wrappers.PRModel) (
	*wrappers.PRResponseModel,
	*wrappers.WebError,
	error,
) {

	return &wrappers.PRResponseModel{
		Message: "PR comment created successfully.",
	}, nil, nil
}
