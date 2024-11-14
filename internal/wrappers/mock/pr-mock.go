package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type PRMockWrapper struct {
}

func (pr *PRMockWrapper) PostPRDecoration(model interface{}) (
	string,
	*wrappers.WebError,
	error,
) {
	switch model.(type) {
	case *wrappers.PRModel:
		return "PR comment created successfully.", nil, nil
	case *wrappers.GitlabPRModel:
		return "MR comment created successfully.", nil, nil
	case *wrappers.BitbucketCloudPRModel:
		return "Bitbucket Cloud PR comment created successfully.", nil, nil
	case *wrappers.BitbucketServerPRModel:
		return "Bitbucket Server PR comment created successfully.", nil, nil
	default:
		return "", nil, errors.New("unsupported model type")
	}
}

func (pr *PRMockWrapper) PostAzurePRDecoration(model *wrappers.AzurePRModel) (string, *wrappers.WebError, error) {
	return "PR comment created successfully.", nil, nil
}
