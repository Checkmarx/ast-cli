// +build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/commands/util"
	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

const fileName = "result-test"

// Create a scan and test getting its results
func TestResultListJson(t *testing.T) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	resultCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(
		resultCommand,
		"result",
		flag(params.TargetFormatFlag), strings.Join([]string{util.FormatJSON, util.FormatSarif, util.FormatSummary}, ","),
		flag(params.TargetFlag), fileName,
		flag(params.ScanIDFlag), scanID,
	)
	assert.NilError(t, err, "Getting results should pass")

	extensions := []string{util.FormatJSON, util.FormatSarif, util.FormatHTML}
	defer func() {
		for _, e := range extensions {
			_ = os.Remove(fmt.Sprintf("%s.%s", fileName, e))
		}
	}()

	result := wrappers.ScanResultsCollection{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, uint(len(result.Results)) == result.TotalCount, "Should have results")

	for _, e := range extensions {
		_, err = os.Stat(fmt.Sprintf("%s.%s", fileName, e))
		assert.NilError(t, err, "Report file should exist for extension "+e)
	}
}
