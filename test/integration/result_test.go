//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
	testifyAssert "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
)

const (
	fileName         = "result-test"
	resultsDirectory = "output-results-folder/"
)

func TestResultsExitCode_OnSendingFakeScanId_ShouldReturnNotFoundError(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	scansPath := viper.GetString(params.ScansPathKey)
	scansWrapper := wrappers.NewHTTPScansWrapper(scansPath)
	results, _ := commands.GetScannerResults(scansWrapper, "FakeScanId", "sast,sca")

	testifyAssert.Nil(t, results, "results should be nil")
}

func TestResultsExitCode_OnSuccessfulScan_ShouldReturnStatusCompleted(t *testing.T) {
	scanID, _ := getRootScan(t)

	scansPath := viper.GetString(params.ScansPathKey)
	scansWrapper := wrappers.NewHTTPScansWrapper(scansPath)
	results, _ := commands.GetScannerResults(scansWrapper, scanID, "sast,sca")

	assert.Equal(t, 1, len(results))
	assert.Equal(t, wrappers.ScanCompleted, (results[0]).Status)
	assert.Equal(t, "", (results[0]).Details)
	assert.Equal(t, "", (results[0]).ErrorCode)
	assert.Equal(t, "", (results[0]).Name)
}

func TestResultsExitCode_NoScanIdSent_FailCommandWithError(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	args := []string{
		"results", "exit-code",
		flag(params.ScanTypes), "sast",
	}

	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, errorConstants.ScanIDRequired)
}

func TestResultsExitCode_FakeScanIdSent_FailCommandWithError(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	args := []string{
		"results", "exit-code",
		flag(params.ScanTypes), "sast",
		flag(params.ScanIDFlag), "FakeScanId",
	}

	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, "Failed showing a scan")
}

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
				printer.FormatIndentedJSON,
				printer.FormatSarif,
				printer.FormatSummary,
				printer.FormatSummaryConsole,
				printer.FormatSonar,
				printer.FormatSummaryJSON,
				printer.FormatPDF,
				printer.FormatSummaryMarkdown,
			}, ",",
		),
		flag(params.TargetFlag), fileName,
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetPathFlag), resultsDirectory,
		flag(params.SastRedundancyFlag),
	)

	result := wrappers.ScanResultsCollection{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, uint(len(result.Results)) == result.TotalCount, "Should have results")

	assertResultFilesCreated(t)

	deleteScanAndProject()
}

// assert all files were created
func assertResultFilesCreated(t *testing.T) {
	extensions := []string{printer.FormatJSON, printer.FormatSarif, printer.FormatHTML, printer.FormatJSON, printer.FormatPDF, printer.FormatMarkdown}

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

func TestResultsGeneratingPdfReportWithInvalidPdfOptions(t *testing.T) {
	scanID, _ := getRootScan(t)

	args := []string{
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "pdf",
		flag(params.ReportFormatPdfOptionsFlag), "invalid_option",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "report option \"invalid_option\" unavailable")
}

func TestResultsGeneratingPdfReportWithInvalidEmail(t *testing.T) {
	scanID, _ := getRootScan(t)

	args := []string{
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "pdf",
		flag(params.ReportFormatPdfToEmailFlag), "valid@mail.com,invalid_email",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "report not sent, invalid email address: invalid_email")
}

func TestResultsGeneratingPdfReportWithPdfOptionsWithoutNotExploitable(t *testing.T) {
	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Results show generating PDF report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "pdf",
		flag(params.ReportFormatPdfOptionsFlag), "Iac-Security,ScanSummary,ExecutiveSummary,ScanResults",
		flag(params.TargetFlag), fileName,
		flag(params.FilterFlag), "state=exclude_not_exploitable",
	)
	defer func() {
		os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
		log.Println("test file removed!")
	}()
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "Report file should exist: "+fileName+printer.FormatPDF)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestResultsGeneratingPdfReportWithPdfOptions(t *testing.T) {
	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Results show generating PDF report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "pdf",
		flag(params.ReportFormatPdfOptionsFlag), "Iac-Security,ScanSummary,ExecutiveSummary,ScanResults",
		flag(params.TargetFlag), fileName,
	)
	defer func() {
		os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
		log.Println("test file removed!")
	}()
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "Report file should exist: "+fileName+printer.FormatPDF)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestResultsGeneratingPdfReportAndSendToEmail(t *testing.T) {
	scanID, _ := getRootScan(t)
	outputBuffer := executeCmdNilAssertion(
		t, "Results show generating PDF report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "pdf",
		flag(params.ReportFormatPdfOptionsFlag), "Iac-Security,ScanSummary,ExecutiveSummary,ScanResults",
		flag(params.ReportFormatPdfToEmailFlag), "test@checkmarx.com,test+2@checkmarx.com",
	)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestResultsGeneratingSBOMWrongScanType(t *testing.T) {
	scanID, _ := getRootScan(t)

	args := []string{
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "sbom",
		flag(params.ReportSbomFormatFlag), "wrong",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Invalid SBOM option")
}

func TestResultsGeneratingSBOMWithProxy(t *testing.T) {
	scanID, _ := getRootScan(t)

	args := []string{
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "sbom",
		flag(params.ReportSbomFormatFlag), "CycloneDxXml",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "TestResultsGeneratingSBOMWithProxy")
}

func TestResultsGeneratingSBOM(t *testing.T) {
	scanID, _ := getRootScan(t)

	args := []string{
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "sbom",
		flag(params.ReportSbomFormatFlag), "CycloneDxXml",
		flag(params.ReportSbomFormatLocalFlowFlag),
	}

	executeCommand(t, args...)
	// not supported yet, no assert to be done
}

func TestResultsWrongScanID(t *testing.T) {
	args := []string{
		"results", "show",
		flag(params.ScanIDFlag), "wrong",
		flag(params.TargetFormatFlag), "json",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "scan not found")
}

func TestResultsCounterJsonOutput(t *testing.T) {
	scanID, _ := getRootScan(t)
	_ = executeCmdNilAssertion(
		t, "Results show generating JSON report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), printer.FormatJSON,
		flag(params.TargetPathFlag), resultsDirectory,
		flag(params.TargetFlag), fileName,
	)

	defer func() {
		_ = os.RemoveAll(fmt.Sprintf(resultsDirectory))
	}()

	result := wrappers.ScanResultsCollection{}

	_, err := os.Stat(fmt.Sprintf("%s%s.%s", resultsDirectory, fileName, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatJSON)

	file, err := os.ReadFile(fmt.Sprintf("%s%s.%s", resultsDirectory, fileName, printer.FormatJSON))
	assert.NilError(t, err, "error reading file")

	err = json.Unmarshal(file, &result)
	assert.NilError(t, err, "error unmarshalling file")

	assert.Assert(t, uint(len(result.Results)) == result.TotalCount, "Should have results")

}

func TestResultsGeneratingJsonReportWithSeverityHighAndWithoutNotExploitable(t *testing.T) {
	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Results show generating Json report with severity and state",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), printer.FormatJSON,
		flag(params.TargetFlag), fileName,
		flag(params.FilterFlag), "state=exclude_not_exploitable, severity=High",
	)
	defer func() {
		os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
		log.Println("test file removed!")
	}()
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist: "+fileName+printer.FormatJSON)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestResultExcludeNotExploitableFailScanId(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	args := []string{
		"results", "show",
		flag(params.ScanIDFlag), "FakeScanId",
		flag(params.TargetFormatFlag), printer.FormatJSON,
		flag(params.TargetFlag), fileName,
		flag(params.FilterFlag), "state=exclude_not_exploitable",
	}
	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, "Failed showing a scan")
}

func TestResultsGeneratingReportWithExcludeNotExploitableStateAndSeverityAndStatus(t *testing.T) {
	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Results show generating Json report with severity and state",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), printer.FormatJSON,
		flag(params.TargetFlag), fileName,
		flag(params.FilterFlag), "state=exclude_not_exploitable;TO_VERIFY,severity=High;Low,status=new;recurrent",
	)
	defer func() {
		os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
		log.Println("test file removed!")
	}()
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist: "+fileName+printer.FormatJSON)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}
