package mock

import (
	"strings"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type JWTMockWrapper struct {
	AIEnabled               int
	CustomGetAllowedEngines func(wrappers.FeatureFlagsWrapper) (map[string]bool, error)
	SscsNewLicensing        bool
}

const AIProtectionDisabled = 1

// GetAllowedEngines mock for tests
func (j *JWTMockWrapper) GetAllowedEngines(featureFlagsWrapper wrappers.FeatureFlagsWrapper) (allowedEngines map[string]bool, sscsNewLicensing bool, err error) {
	if j.CustomGetAllowedEngines != nil {
		allowedEngines, err = j.CustomGetAllowedEngines(featureFlagsWrapper)
		return allowedEngines, j.SscsNewLicensing, err
	}
	allowedEngines = make(map[string]bool)
	engines := []string{"sast", "iac-security", "sca", "api-security", "containers", "scs"}
	for _, value := range engines {
		allowedEngines[strings.ToLower(value)] = true
	}
	return allowedEngines, j.SscsNewLicensing, nil
}

func (*JWTMockWrapper) ExtractTenantFromToken() (tenant string, err error) {
	return "test-tenant", nil
}

// IsAllowedEngine mock for tests
func (j *JWTMockWrapper) IsAllowedEngine(engine string) (bool, error) {
	if j.AIEnabled == AIProtectionDisabled {
		return false, nil
	}
	return true, nil
}
