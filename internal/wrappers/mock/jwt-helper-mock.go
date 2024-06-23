package mock

import (
	"strings"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type JWTMockWrapper struct{}

// GetAllowedEngines mock for tests
func (*JWTMockWrapper) GetAllowedEngines(featureFlagsWrapper wrappers.FeatureFlagsWrapper) (allowedEngines map[string]bool, err error) {
	allowedEngines = make(map[string]bool)
	engines := []string{"sast", "iac-security", "sca", "api-security", "scs"}
	for _, value := range engines {
		allowedEngines[strings.ToLower(value)] = true
	}
	return allowedEngines, nil
}

func (*JWTMockWrapper) ExtractTenantFromToken() (tenant string, err error) {
	return "test-tenant", nil
}
