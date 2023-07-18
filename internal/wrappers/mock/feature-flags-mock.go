package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"strings"
)

type FeatureFlagsMockWrapper struct{}

func (f FeatureFlagsMockWrapper) GetAll() (*wrappers.FeatureFlagsResponseModel, error) {
	allowedEngines := make(map[string]bool)
	engines := []string{"sast", "iac-security", "sca", "api-security"}
	for _, value := range engines {
		allowedEngines[strings.ToLower(value)] = true
	}
	return &wrappers.FeatureFlagsResponseModel{}, nil
}
