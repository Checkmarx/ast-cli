package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

var Flags wrappers.FeatureFlagsResponseModel
var Flag wrappers.FeatureFlagResponseModel

type FeatureFlagsMockWrapper struct {
}

func (f FeatureFlagsMockWrapper) GetAll() (*wrappers.FeatureFlagsResponseModel, error) {
	fmt.Println("Called GetAll in FeatureFlagsMockWrapper")
	return &Flags, nil
}

func (f FeatureFlagsMockWrapper) GetSpecificFlag(specificFlag string) (*wrappers.FeatureFlagResponseModel, error) {
	fmt.Println("Called GetSpecificFlag in FeatureFlagsMockWrapper with flag:", specificFlag)
	return &Flag, nil
}
