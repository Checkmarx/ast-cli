package commands

import (
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	allowAiUsageKey         = "scan.config.ai.allowAiUsage"
	sastFindingsAnalysisKey = "scan.config.sast.findingsAnalysis"
	sastExtendedAnalysisKey = "scan.config.sast.extendedAnalysis"
)

// resolveDefaultIncludeFilters returns the default include filter for the source zip. The Linguist patterns are
// added only when AI usage is allowed and LLM-based scanning is enabled at tenant or project level; any failure to
// read the settings falls back to the SAST patterns so the scan still runs as before.
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
	if isLlmScanningOn(tenantSettings) {
		logger.PrintIfVerbose("LLM-based scanning is enabled at tenant level.")
		return true
	}
	if projectID == "" {
		return false
	}
	projectConfig, projectErrorModel, err := projectsWrapper.GetConfiguration(projectID)
	if err != nil || projectErrorModel != nil {
		logger.PrintfIfVerbose("Failed to read the project configuration, LLM-based scanning treated as disabled: %v %v", err, projectErrorModel)
		return false
	}
	if isLlmScanningOn(projectSettingsMap(projectConfig)) {
		logger.PrintIfVerbose("LLM-based scanning is enabled at project level.")
		return true
	}
	return false
}

// isLlmScanningOn reports whether both LLM-based settings (findings analysis and LLM-based scanning) are on.
func isLlmScanningOn(settings map[string]string) bool {
	return isSettingTrue(settings, sastFindingsAnalysisKey) && isSettingTrue(settings, sastExtendedAnalysisKey)
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
