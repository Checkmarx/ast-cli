package mock

import (
	"strings"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type FeatureFlagsMockWrapper struct{}

func (f FeatureFlagsMockWrapper) GetAll(string) (*wrappers.FeatureFlagsResponseModel, error) {
	allowedEngines := make(map[string]bool)
	engines := []string{"sast", "iac-security", "sca", "api-security"}
	for _, value := range engines {
		allowedEngines[strings.ToLower(value)] = true
	}
	return &wrappers.FeatureFlagsResponseModel{}, nil
}
