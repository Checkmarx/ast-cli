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
	tenantConfig, errorModel, err := tenantWrapper.GetTenantConfiguration()
	if err != nil || errorModel != nil {
		logger.PrintfIfVerbose("Failed to read the tenant configuration, LLM-based scanning treated as disabled: %v %v", err, errorModel)
		return false
	}
	tenantSettings := tenantSettingsMap(tenantConfig)
	if !isSettingTrue(tenantSettings, allowAiUsageKey) {
		logger.PrintIfVerbose("Allow AI Usage is off at tenant level.")
		return false
	}
	if projectID != "" {
		if enabled, ok := projectLlmScanningSetting(projectsWrapper, projectID); ok {
			logger.PrintfIfVerbose("LLM-based scanning is %s at project level.", enabledText(enabled))
			return enabled
		}
	}
	enabled := isLlmScanningOn(tenantSettings)
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
	settings := projectSettingsMap(projectConfig)
	if strings.TrimSpace(settings[sastExtendedAnalysisKey]) == "" {
		return false, false
	}
	return isLlmScanningOn(settings), true
}

func enabledText(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

// isLlmScanningOn reports whether the LLM-based scanning setting (scan.config.sast.extendedAnalysis) is on.
func isLlmScanningOn(settings map[string]string) bool {
	return isSettingTrue(settings, sastExtendedAnalysisKey)
}

func isSettingTrue(settings map[string]string, key string) bool {
	return strings.EqualFold(strings.TrimSpace(settings[key]), "true")
}

func tenantSettingsMap(config *[]*wrappers.TenantConfigurationResponse) map[string]string {
	settings := map[string]string{}
	if config == nil {
		return settings
	}
	for _, c := range *config {
		if c != nil {
			settings[c.Key] = c.Value
		}
	}
	return settings
}

func projectSettingsMap(config []wrappers.ProjectConfiguration) map[string]string {
	settings := make(map[string]string, len(config))
	for _, c := range config {
		settings[c.Key] = c.Value
	}
	return settings
}
