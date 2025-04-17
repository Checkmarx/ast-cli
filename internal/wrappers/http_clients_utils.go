package wrappers

import (
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
)

const scanOverviewDefaultPath = "api/micro-engines/read/scans/%s/scan-overview"
const riskManagementDefaultPath = "api/risk-management/projects/%s/results?scanID=%s"

func setDefaultPath(path, defaultPath, pathKey string) string {
	if path != defaultPath {
		configFilePath, err := configuration.GetConfigFilePath()
		if err != nil {
			logger.PrintfIfVerbose("Error getting config file path: %v", err)
		}
		err = configuration.SafeWriteSingleConfigKeyString(configFilePath, pathKey, defaultPath)
		if err != nil {
			logger.PrintfIfVerbose("Error writing %s path key to config file. error: %v", pathKey, err)
		}
	}
	return defaultPath
}
