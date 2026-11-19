package commands

import (
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	// allowAiUsageKey is the tenant setting that must be on before the LLM-based scanning settings are checked.
	allowAiUsageKey = "scan.config.ai.allowAiUsage"
	// sastExtendedAnalysisKey is the LLM-based scanning setting, available at tenant and project level.
	sastExtendedAnalysisKey = "scan.config.sast.extendedAnalysis"
)

// resolveDefaultIncludeFilters returns the default include filter for the source zip. The Linguist patterns are
// added only when Allow AI Usage is on and LLM-based scanning is enabled, with the project setting overriding the
// tenant setting; any failure to read the tenant settings falls back to the SAST patterns so the scan runs as before.
func resolveDefaultIncludeFilters(
	tenantWrapper wrappers.TenantConfigurationWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	projectID string,
) []string {
	enabled := isLlmScanningEnabled(tenantWrapper, projectsWrapper, projectID)
	if enabled {
		logger.PrintIfVerbose("LLM-based scanning is enabled: adding Linguist patterns to the default include filter.")
	} else {
		logger.PrintIfVerbose("LLM-based scanning is disabled: using the default SAST include filter only.")
	}
	return commonParams.DefaultIncludeFilters(enabled)
}

// isLlmScanningEnabled reports whether LLM-based scanning is enabled for the scan.
func isLlmScanningEnabled(tenantWrapper wrappers.TenantConfigurationWrapper, projectsWrapper wrappers.ProjectsWrapper, projectID string) bool {
	tenantConfig, err := GetTenantConfigurationResponses(tenantWrapper)
	if err != nil {
		logger.PrintfIfVerbose("Failed to read the tenant configuration, LLM-based scanning treated as disabled: %v", err)
		return false
	}
	if !isTenantSettingTrue(tenantConfig, allowAiUsageKey) {
		logger.PrintIfVerbose("Allow AI Usage is off at tenant level.")
		return false
	}
	if projectID != "" {
		if enabled, ok := projectLlmScanningSetting(projectsWrapper, projectID); ok {
			logger.PrintfIfVerbose("LLM-based scanning is %s at project level.", enabledText(enabled))
			return enabled
		}
	}
	enabled := isTenantSettingTrue(tenantConfig, sastExtendedAnalysisKey)
	logger.PrintfIfVerbose("LLM-based scanning is %s at tenant level.", enabledText(enabled))
	return enabled
}

// projectLlmScanningSetting returns the project's LLM-based scanning setting; ok is false when the project does not
// set it or its settings cannot be read.
func projectLlmScanningSetting(projectsWrapper wrappers.ProjectsWrapper, projectID string) (enabled, ok bool) {
	projectConfig, errorModel, err := projectsWrapper.GetConfiguration(projectID)
	if err != nil || errorModel != nil {
		logger.PrintfIfVerbose("Failed to read the project configuration, using the tenant setting: %v %v", err, errorModel)
		return false, false
	}
	value := projectSettingsMap(projectConfig)[sastExtendedAnalysisKey]
	if strings.TrimSpace(value) == "" {
		return false, false
	}
	return isTrueValue(value), true
}

func enabledText(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

// isTenantSettingTrue reports whether a tenant setting is on; a missing setting is off.
func isTenantSettingTrue(tenantConfig *[]*wrappers.TenantConfigurationResponse, key string) bool {
	value, _ := GetTenantConfiguration(tenantConfig, key)
	return isTrueValue(value)
}

// isTrueValue reports whether a setting value is "true", ignoring case and surrounding spaces.
func isTrueValue(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func projectSettingsMap(config []wrappers.ProjectConfiguration) map[string]string {
	settings := make(map[string]string, len(config))
	for _, c := range config {
		settings[c.Key] = c.Value
	}
	return settings
}
