package mock

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type UploadsMockWrapper struct {
}

func (u *UploadsMockWrapper) UploadFileInMultipart(filePath string, wrapper wrappers.FeatureFlagsWrapper) (*string, error) {
	fmt.Println("UploadFileInMultipart called Create in UploadsMockWrapper")
	if filePath == "failureCase2.zip" {
		return nil, errors.New("error from UploadFileInMultipart")
	}
	url := "/path/to/largeZipFile"
	return &url, nil
}

func (u *UploadsMockWrapper) UploadFile(filePath string, featureFlagsWrapper wrappers.FeatureFlagsWrapper) (*string, error) {
	fmt.Println("Called Create in UploadsMockWrapper")
	if strings.Contains(filePath, "failureCase.zip") {
		return nil, errors.New("error from UploadFile")
	}
	url := "/path/to/nowhere"
	return &url, nil
}
