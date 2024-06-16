package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type UploadsMockWrapper struct {
}

func (u *UploadsMockWrapper) UploadFile(_ string, featureFlagsWrapper wrappers.FeatureFlagsWrapper) (*string, error) {
	fmt.Println("Called Create in UploadsMockWrapper")
	url := "/path/to/nowhere"
	return &url, nil
}
