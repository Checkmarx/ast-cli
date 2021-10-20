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
	scanID, _ := getRootScan(t)
	resultCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(
		resultCommand,
		"result",
		flag(params.TargetFormatFlag), strings.Join([]string{util.FormatJSON, util.FormatSarif, util.FormatSummary, util.FormatSummaryConsole}, ","),
		flag(params.TargetFlag), fileName,
		flag(params.TargetPathFlag), resultsDirectory,
	)
	assertError(t, err, "Please provide a scan ID")

	err = execute(
		resultCommand,
		"result",
		flag(params.TargetFormatFlag), strings.Join([]string{util.FormatJSON, util.FormatSarif, util.FormatSummary, util.FormatSummaryConsole}, ","),
		flag(params.TargetFlag), fileName,
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetPathFlag), resultsDirectory,
	)
	assert.NilError(t, err, "Getting results should pass")

	extensions := []string{util.FormatJSON, util.FormatSarif, util.FormatHTML}
	defer func() {
		_ = os.RemoveAll(fmt.Sprintf(resultsDirectory))
	}()

	result := wrappers.ScanResultsCollection{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, uint(len(result.Results)) == result.TotalCount, "Should have results")

	for _, e := range extensions {
		_, err = os.Stat(fmt.Sprintf("%s%s.%s", resultsDirectory, fileName, e))
		assert.NilError(t, err, "Report file should exist for extension "+e)
	}
}
