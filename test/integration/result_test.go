//go:build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

const (
	fileName         = "result-test"
	resultsDirectory = "output-results-folder/"
)

// Create a scan and test getting its results
func TestResultListJson(t *testing.T) {

	assertRequiredParameter(t, "Please provide a scan ID", "result")

	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(t, "Getting results should pass",
		"result",
		flag(params.TargetFormatFlag), strings.Join([]string{util.FormatJSON, util.FormatSarif, util.FormatSummary, util.FormatSummaryConsole, util.FormatSonar}, ","),
		flag(params.TargetFlag), fileName,
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetPathFlag), resultsDirectory,
	)

	result := wrappers.ScanResultsCollection{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, uint(len(result.Results)) == result.TotalCount, "Should have results")

	assertResultFilesCreated(t)
}

// assert all files were created
func assertResultFilesCreated(t *testing.T) {
	extensions := []string{util.FormatJSON, util.FormatSarif, util.FormatHTML,util.FormatJSON}

	for _, e := range extensions {
		_, err := os.Stat(fmt.Sprintf("%s%s.%s", resultsDirectory, fileName, e))
		assert.NilError(t, err, "Report file should exist for extension "+e)
	}

	// delete directory in the end
	defer func() {
		_ = os.RemoveAll(fmt.Sprintf(resultsDirectory))
	}()
}
