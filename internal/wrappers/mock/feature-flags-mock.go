package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

var Flags wrappers.FeatureFlagsResponseModel

type FeatureFlagsMockWrapper struct {
}

func (f FeatureFlagsMockWrapper) GetAll() (*wrappers.FeatureFlagsResponseModel, error) {
	fmt.Println("Called GetAll in FeatureFlagsMockWrapper")
	return &Flags, nil
}
