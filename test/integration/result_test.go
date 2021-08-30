// +build integration

package integration

import (
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/checkmarxDev/ast-cli/internal/commands/util"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

// Create a scan and test getting its results
func TestResultListJson(t *testing.T) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	resultCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(resultCommand,
		"result",
		flag(commands.FormatFlag), util.FormatJSON,
		flag(commands.ScanIDFlag), scanID,
	)
	assert.NilError(t, err, "Getting results should pass")

	result := wrappers.ScanResultsCollection{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, uint(len(result.Results)) == result.TotalCount, "Should have results")
}

// Create a scan and test getting its results
func TestResultListSarif(t *testing.T) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	resultCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(resultCommand,
		"result",
		flag(commands.TargetFormatFlag), util.FormatSarif,
		flag(commands.ScanIDFlag), scanID,
	)
	assert.NilError(t, err, "Getting results should pass")

	result := wrappers.SarifResultsCollection{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, len(result.Runs) > 0, "Should have results")
}
