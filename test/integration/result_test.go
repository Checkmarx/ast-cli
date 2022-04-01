//go:build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
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

	outputBuffer := executeCmdNilAssertion(
		t, "Getting results should pass",
		"results",
		"show",
		flag(params.TargetFormatFlag), strings.Join(
			[]string{
				printer.FormatJSON,
				printer.FormatSarif,
				printer.FormatSummary,
				printer.FormatSummaryConsole,
				printer.FormatSonar,
				printer.FormatSummaryJSON,
			}, ",",
		),
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
	extensions := []string{printer.FormatJSON, printer.FormatSarif, printer.FormatHTML, printer.FormatJSON}

	for _, e := range extensions {
		_, err := os.Stat(fmt.Sprintf("%s%s.%s", resultsDirectory, fileName, e))
		assert.NilError(t, err, "Report file should exist for extension "+e)
	}

	// delete directory in the end
	defer func() {
		_ = os.RemoveAll(fmt.Sprintf(resultsDirectory))
	}()
}

func TestResultsShowParamFailed(t *testing.T) {

	args := []string{
		"results",
		"show",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Failed listing results: Please provide a scan ID")
}

func TestCodeBashingParamFailed(t *testing.T) {

	args := []string{
		"results",
		"codebashing",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "required flag(s) \"cwe-id\", \"language\", \"vulnerability-type\" not set")
}

func TestCodeBashingList(t *testing.T) {

	outputBuffer := executeCmdNilAssertion(
		t,
		"Getting results should pass",
		"results",
		"codebashing",
		flag(params.LanguageFlag), "PHP",
		flag(params.VulnerabilityTypeFlag), "Reflected XSS All Clients",
		flag(params.CweIDFlag), "79")

	codebashing := []wrappers.CodeBashingCollection{}

	_ = unmarshall(t, outputBuffer, &codebashing, "Reading results should pass")

	assert.Assert(t, codebashing != nil, "Should exist codebashing link")
}

func TestCodeBashingListJson(t *testing.T) {

	outputBuffer := executeCmdNilAssertion(
		t,
		"Getting results should pass",
		"results",
		"codebashing",
		flag(params.LanguageFlag), "PHP",
		flag(params.VulnerabilityTypeFlag), "Reflected XSS All Clients",
		flag(params.CweIDFlag), "79",
		flag(params.FormatFlag), "json")

	codebashing := []wrappers.CodeBashingCollection{}

	_ = unmarshall(t, outputBuffer, &codebashing, "Reading results should pass")

	assert.Assert(t, codebashing != nil, "Should exist codebashing link")
}

func TestCodeBashingListTable(t *testing.T) {

	outputBuffer := executeCmdNilAssertion(
		t,
		"Getting results should pass",
		"results",
		"codebashing",
		flag(params.LanguageFlag), "PHP",
		flag(params.VulnerabilityTypeFlag), "Reflected XSS All Clients",
		flag(params.CweIDFlag), "79",
		flag(params.FormatFlag), "table")

	assert.Assert(t, outputBuffer != nil, "Should exist codebashing link")
}

func TestCodeBashingListEmpty(t *testing.T) {

	args := []string{
		"results",
		"codebashing",
		flag(params.LanguageFlag), "PHP",
		flag(params.VulnerabilityTypeFlag), "Reflected XSS All Clients",
		flag(params.CweIDFlag), "11",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "No codebashing link available")
}

func TestCodeBashingFailedListingAuth(t *testing.T) {

	args := []string{
		"results",
		"codebashing",
		flag(params.LanguageFlag), "PHP",
		flag(params.VulnerabilityTypeFlag), "Reflected XSS All Clients",
		flag(params.CweIDFlag), "11",
		flag(params.AccessKeySecretFlag), "mock",
		flag(params.AccessKeyIDFlag), "mock",
		flag(params.AstAPIKeyFlag), "mock",
	}
	err, _ := executeCommand(t, args...)
	assertError(t, err, "Authentication failed, not able to retrieve codebashing base link")
}
