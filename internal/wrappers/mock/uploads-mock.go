package mock

import (
	"fmt"
	"github.com/pkg/errors"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type UploadsMockWrapper struct {
}

func (u *UploadsMockWrapper) UploadFile(filePath string, featureFlagsWrapper wrappers.FeatureFlagsWrapper) (*string, error) {
	fmt.Println("Called Create in UploadsMockWrapper")
	if filePath == "failureCase.zip" {
		return nil, errors.New("error from UploadFile")
	}
	url := "/path/to/nowhere"
	return &url, nil
}
