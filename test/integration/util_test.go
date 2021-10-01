// +build integration

package integration

import (
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestUtilLogsSAST(t *testing.T) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	logsCommand := createASTIntegrationTestCommand(t)

	err := execute(
		logsCommand,
		"utils", "logs",
		flag(params.ScanIDFlag), scanID,
		flag(params.ScanTypeFlag), "sast",
	)
	assert.NilError(t, err, "Getting scan SAST log should pass")
}

func TestUtilLogsKICS(t *testing.T) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	logsCommand := createASTIntegrationTestCommand(t)

	err := execute(
		logsCommand,
		"utils", "logs",
		flag(params.ScanIDFlag), scanID,
		flag(params.ScanTypeFlag), "kics",
	)
	assert.NilError(t, err, "Getting scan KICS log should pass")
}
