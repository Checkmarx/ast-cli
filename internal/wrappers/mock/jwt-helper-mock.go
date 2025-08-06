package mock

import (
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type JWTMockWrapper struct {
	AIEnabled                 int
	CheckmarxOneAssistEnabled int
	CustomGetAllowedEngines   func(wrappers.FeatureFlagsWrapper) (map[string]bool, error)
}

const AIProtectionDisabled = 1
const CheckmarxOneAssistDisabled = 1

var engines = []string{"sast", "sca", "api-security", "iac-security", "scs", "containers", "enterprise-secrets"}

// GetAllowedEngines mock for tests
func (j *JWTMockWrapper) GetAllowedEngines(featureFlagsWrapper wrappers.FeatureFlagsWrapper) (allowedEngines map[string]bool, err error) {
	if j.CustomGetAllowedEngines != nil {
		return j.CustomGetAllowedEngines(featureFlagsWrapper)
	}
	allowedEngines = make(map[string]bool)

	for _, value := range engines {
		allowedEngines[strings.ToLower(value)] = true
	}
	return allowedEngines, nil
}

func (*JWTMockWrapper) ExtractTenantFromToken() (tenant string, err error) {
	return "test-tenant", nil
}

// IsAllowedEngine mock for tests
func (j *JWTMockWrapper) IsAllowedEngine(engine string) (bool, error) {
	if engine == params.AiProviderFlag || engine == params.EnterpriseSecretsLabel {
		if j.AIEnabled == AIProtectionDisabled {
			return false, nil
		}
		return true, nil
	}

	if engine == params.CheckmarxOneAssistType {
		if j.CheckmarxOneAssistEnabled == CheckmarxOneAssistDisabled {
			return false, nil
		}
		return true, nil
	}
	return true, nil
}
