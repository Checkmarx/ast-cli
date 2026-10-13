package util

import (
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
)

// SendRealtimeScanTelemetry posts one telemetry event for a completed realtime scan.
// githubUrl is the GitHub origin of the scanned file's directory. A nil wrapper is a no-op.
func SendRealtimeScanTelemetry(telemetry wrappers.TelemetryWrapper, engine, scanType, fileSource string, totalCount int) {
	if telemetry == nil {
		return
	}
	err := telemetry.SendAIDataToLog(&wrappers.DataForAITelemetry{
		AIProvider:       utils.GetOptionalParam("aiProvider"),
		Agent:            utils.GetOptionalParam("agent"),
		Engine:           engine,
		ScanType:         scanType,
		Type:             "scan",
		SubType:          "scan",
		TotalCount:       totalCount,
		UniqueID:         wrappers.GetUniqueID(),
		AiAgentSessionId: utils.GetOptionalParam("aiAgentSessionId"),
		GithubURL:        utils.GitHubRemoteURL(scanWorkDir(fileSource)),
	})
	if err != nil {
		logger.PrintfIfVerbose("Failed to send telemetry: %v", err)
	}
}

func scanWorkDir(fileSource string) string {
	fileSource = strings.TrimSpace(fileSource)
	if fileSource == "" {
		return ""
	}
	first := strings.TrimSpace(strings.Split(fileSource, ",")[0])
	if first == "" {
		return ""
	}
	return filepath.Dir(first)
}
