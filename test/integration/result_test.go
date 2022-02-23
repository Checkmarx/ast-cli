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

	assertRequiredParameter(t, "Please provide a scan ID", "results", "show")

	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(t, "Getting results should pass",
		"results",
		"show",
		flag(params.TargetFormatFlag), strings.Join([]string{util.FormatJSON, util.FormatSarif, util.FormatSummary, util.FormatSummaryConsole, util.FormatSonar, util.FormatSummaryJSON}, ","),
		flag(params.TargetFlag), fileName,
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetPathFlag), resultsDirectory,
	)

	result := wrappers.ScanResultsCollection{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, uint(len(result.Results)) == result.TotalCount, "Should have results")

	assertResultFilesCreated(t)

	deleteScanAndProject()
}

// assert all files were created
func assertResultFilesCreated(t *testing.T) {
	extensions := []string{util.FormatJSON, util.FormatSarif, util.FormatHTML, util.FormatJSON}

	for _, e := range extensions {
		_, err := os.Stat(fmt.Sprintf("%s%s.%s", resultsDirectory, fileName, e))
		assert.NilError(t, err, "Report file should exist for extension "+e)
	}

	// delete directory in the end
	defer func() {
		_ = os.RemoveAll(fmt.Sprintf(resultsDirectory))
	}()
}

func TestCodeBashingParamFailed(t *testing.T) {

	args := []string{
		"results",
		"codebashing",
		flag(params.LanguageFlag), "PHP",
		flag(params.VulnerabilityTypeFlag), "'Reflected XSS All Clients'",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "required flag(s) \"cwe-id\" not set")
}

func TestCodeBashingList(t *testing.T) {

	outputBuffer := executeCmdNilAssertion(
		t,
		"Getting results should pass",
		"results",
		"codebashing",
		flag(params.LanguageFlag), "PHP",
		flag(params.VulnerabilityTypeFlag), "Reflected XSS All Clients",
		flag(params.CweIDFlag), "79",)

	codebashing := []wrappers.CodeBashingCollection{}

	_ = unmarshall(t, outputBuffer, &codebashing, "Reading results should pass")

	assert.Assert(t,codebashing!=nil, "Should exist codebashing link")
}