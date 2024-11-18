package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

const prCommentSuccess = "PR comment created successfully."

type PRMockWrapper struct {
}

func (pr *PRMockWrapper) PostPRDecoration(model interface{}) (
	string,
	*wrappers.WebError,
	error,
) {
	switch model.(type) {
	case *wrappers.PRModel:
		return prCommentSuccess, nil, nil
	case *wrappers.GitlabPRModel:
		return "MR comment created successfully.", nil, nil
	case *wrappers.BitbucketCloudPRModel:
		return "Bitbucket Cloud PR comment created successfully.", nil, nil
	case *wrappers.BitbucketServerPRModel:
		return "Bitbucket Server PR comment created successfully.", nil, nil
	case *wrappers.AzurePRModel:
		return prCommentSuccess, nil, nil

	default:
		return "", nil, errors.New("unsupported model type")
	}
}
