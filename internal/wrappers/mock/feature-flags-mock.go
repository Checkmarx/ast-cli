package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type FeatureFlagsMockWrapper struct{}

func (f FeatureFlagsMockWrapper) GetAll() (*wrappers.FeatureFlagsResponseModel, error) {
	return &wrappers.FeatureFlagsResponseModel{}, nil
}
