package wrappers

import (
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
)

const scanOverviewDefaultPath = "api/micro-engines/read/scans/%s/scan-overview"
const riskManagementDefaultPath = "api/risk-management/projects/%s/results?scanID=%s"

// setDefaultPath checks if the path is the default path, if not it writes the default path to the config file
func setDefaultPath(path, defaultPath string) string {
	if path != defaultPath {
		configFilePath, err := configuration.GetConfigFilePath()
		if err != nil {
			logger.PrintfIfVerbose("Error getting config file path: %v", err)
		}
		err = configuration.SafeWriteSingleConfigKeyString(configFilePath, commonParams.ScsScanOverviewPathKey, defaultPath)
		if err != nil {
			logger.PrintfIfVerbose("Error writing Scan Overview path to config file: %v", err)
		}
	}
	return defaultPath
}
