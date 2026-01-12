package mock

import (
	"strconv"
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type JWTMockWrapper struct {
	AIEnabled                 int
	EnterpriseSecretsEnabled  int
	SecretDetectionEnabled    int
	CheckmarxOneAssistEnabled int
	DastEnabled               bool
	TenantName                string
	AllowedEngines            []string
	CustomGetAllowedEngines   func(wrappers.FeatureFlagsWrapper) (map[string]bool, error)
}

const AIProtectionDisabled = 1
const CheckmarxOneAssistDisabled = 1
const EnterpriseSecretsDisabled = 1
const SecretDetectionDisabled = 1

var engines = []string{"sast", "sca", "api-security", "iac-security", "scs", "containers", "enterprise-secrets"}

const licenseEnabledValue = "true"

// GetAllowedEngines mock for tests
func (j *JWTMockWrapper) GetAllowedEngines(featureFlagsWrapper wrappers.FeatureFlagsWrapper) (allowedEngines map[string]bool, err error) {
	if j.CustomGetAllowedEngines != nil {
		allowedEngines, err = j.CustomGetAllowedEngines(featureFlagsWrapper)
		return allowedEngines, err
	}
	allowedEngines = make(map[string]bool)
	enginesToCopy := engines
	if j.AllowedEngines != nil {
		enginesToCopy = j.AllowedEngines
	}

	for _, value := range enginesToCopy {
		allowedEngines[strings.ToLower(value)] = true
	}
	return allowedEngines, nil
}

func (j *JWTMockWrapper) ExtractTenantFromToken() (tenant string, err error) {
	return j.TenantName, nil
}

// IsAllowedEngine mock for tests
func (j *JWTMockWrapper) IsAllowedEngine(engine string) (bool, error) {
	if engine == params.AiProviderFlag {
		if j.AIEnabled == AIProtectionDisabled {
			return false, nil
		}
		return true, nil
	}

	if engine == params.EnterpriseSecretsLabel {
		if j.EnterpriseSecretsEnabled == EnterpriseSecretsDisabled {
			return false, nil
		}
		return true, nil
	}

	if engine == params.SecretDetectionLabel {
		if j.SecretDetectionEnabled == SecretDetectionDisabled {
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

func (j *JWTMockWrapper) CheckPermissionByAccessToken(requiredPermission string) (permission bool, err error) {
	return true, nil
}

// IsDastEnabled mock for tests
func (j *JWTMockWrapper) IsDastEnabled() (bool, error) {
	return j.DastEnabled, nil
}

// GetAllJwtClaims mock for tests
func (j *JWTMockWrapper) GetAllJwtClaims() (*wrappers.JwtClaims, error) {
	return &wrappers.JwtClaims{
		TenantName:     j.TenantName,
		DastEnabled:    j.DastEnabled,
		AllowedEngines: j.AllowedEngines,
	}, nil
}

func (j *JWTMockWrapper) GetLicenseDetails() (licenseDetails map[string]string, err error) {
	licenseDetails = make(map[string]string)

	assistEnabled := (j.CheckmarxOneAssistEnabled != CheckmarxOneAssistDisabled) || (j.AIEnabled != AIProtectionDisabled)
	licenseDetails["scan.config.plugins.cxoneassist"] = strconv.FormatBool(assistEnabled)

	standaloneEnabled := true
	licenseDetails["scan.config.plugins.cxdevassist"] = strconv.FormatBool(standaloneEnabled)

	for _, engine := range engines {
		licenseDetails[engine] = licenseEnabledValue
	}

	return licenseDetails, nil
}
