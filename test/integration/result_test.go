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
	fileExtention    = "report.json"

	//----------------------------------------------------------------------------------------------------------------------
	// This ScanIDWithDevAndTestDep is associated with the CXOne project: ASTCLI/HideDevAndTestsVulnerabilities/Test (DEU, Galactica tenant).
	// All vulnerable packages in this project have been snoozed or muted, so no vulnerabilities should appear in this scan.
	// If the test fails, verify the scan exists in this project. If it doesn't, create a new scan for the project using
	// DevAndTestsVulnerabilitiesProject.zip, mute and snooze all packages, and update the scanID accordingly.
	ScanIDWithDevAndTestDep = "28d29a61-bc5e-4f5a-9fdd-e18c5a10c05b"
	//----------------------------------------------------------------------------------------------------------------------
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

func TestResultListForGlReports(t *testing.T) {
	assertRequiredParameter(t, "Please provide a scan ID", "results", "show")

	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Getting results should pass",
		"results",
		"show",
		"--debug",
		flag(params.TargetFormatFlag), strings.Join(
			[]string{
				printer.FormatGLSast,
				printer.FormatGLSca,
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

	assertGlResultFilesCreated(t)
}

func assertGlResultFilesCreated(t *testing.T) {
	extensions := []string{printer.FormatGLSast,
		printer.FormatGLSca}

	for _, e := range extensions {
		_, err := os.Stat(fmt.Sprintf("%s%s.%s-%s", resultsDirectory, fileName, e, fileExtention))
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

func TestResultsGeneratingJsonCxOneReportWithInvalidJsonOptions(t *testing.T) {
	scanID, _ := getRootScan(t)

	args := []string{
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "json-cxOne",
		flag(params.ReportFormatJsonOptionsFlag), "invalid_option",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "report option \"invalid_option\" unavailable")
}

func TestResultsGeneratingJsonCxOneReportWithInvalidEmail(t *testing.T) {
	scanID, _ := getRootScan(t)

	args := []string{
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "json-cxOne",
		flag(params.ReportFormatJsonToEmailFlag), "valid@mail.com,invalid_email",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "report not sent, invalid email address: invalid_email")
}

func TestResultsGeneratingJsonCxOneReportWithJsonOptionsWithoutNotExploitable(t *testing.T) {
	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Results show generating json cxOne report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "json-cxOne",
		flag(params.ReportFormatJsonOptionsFlag), "Iac-Security,ScanSummary,ExecutiveSummary,ScanResults",
		flag(params.TargetFlag), fileName,
		flag(params.FilterFlag), "state=exclude_not_exploitable",
	)
	defer func() {
		os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
		log.Println("test file removed!")
	}()
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist: "+fileName+printer.FormatJSON)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestResultsGeneratingJsonCxOneReportWithJsonOptions(t *testing.T) {
	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Results show generating json cxOne report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "json-cxOne",
		flag(params.ReportFormatJsonOptionsFlag), "Iac-Security,ScanSummary,ExecutiveSummary,ScanResults",
		flag(params.TargetFlag), fileName,
	)
	defer func() {
		os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
		log.Println("test file removed!")
	}()
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist: "+fileName+printer.FormatJSON)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestResultsGeneratingJsonCxOneReportAndSendToEmail(t *testing.T) {
	scanID, _ := getRootScan(t)
	outputBuffer := executeCmdNilAssertion(
		t, "Results show generating json cxOne report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), "json-cxOne",
		flag(params.ReportFormatJsonOptionsFlag), "Iac-Security,ScanSummary,ExecutiveSummary,ScanResults",
		flag(params.ReportFormatJsonToEmailFlag), "test@checkmarx.com,test+2@checkmarx.com",
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

func TestResultsCounterGlSastOutput(t *testing.T) {
	scanID, _ := getRootScan(t)
	_ = executeCmdNilAssertion(
		t, "Results show generating gl-sast report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), printer.FormatGLSast,
		flag(params.TargetPathFlag), resultsDirectory,
		flag(params.TargetFlag), fileName,
	)

	defer func() {
		_ = os.RemoveAll(fmt.Sprintf(resultsDirectory))
	}()

	result := wrappers.ScanResultsCollection{}

	_, err := os.Stat(fmt.Sprintf("%s%s.%s-%s", resultsDirectory, fileName, printer.FormatGLSast, fileExtention))

	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatGLSast)

	file, err := os.ReadFile(fmt.Sprintf("%s%s.%s-%s", resultsDirectory, fileName, printer.FormatGLSast, fileExtention))
	assert.NilError(t, err, "error reading file")

	err = json.Unmarshal(file, &result)
	assert.NilError(t, err, "error unmarshalling file")

	assert.Assert(t, uint(len(result.Results)) == result.TotalCount, "Should have results")

}

func TestResultsCounterGlSCAOutput(t *testing.T) {
	scanID, _ := getRootScan(t)
	_ = executeCmdNilAssertion(
		t, "Results show generating gl-sca report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), printer.FormatGLSca,
		flag(params.TargetPathFlag), resultsDirectory,
		flag(params.TargetFlag), fileName,
	)

	defer func() {
		_ = os.RemoveAll(fmt.Sprintf(resultsDirectory))
	}()

	result := wrappers.ScanResultsCollection{}

	_, err := os.Stat(fmt.Sprintf("%s%s.%s-%s", resultsDirectory, fileName, printer.FormatGLSca, fileExtention))

	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatGLSca)

	file, err := os.ReadFile(fmt.Sprintf("%s%s.%s-%s", resultsDirectory, fileName, printer.FormatGLSca, fileExtention))
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

func TestResultsGeneratingReportWithExcludeNotExploitableUpperCaseStateAndSeverityAndStatus(t *testing.T) {
	scanID, _ := getRootScan(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Results show generating Json report with severity and state",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), printer.FormatJSON,
		flag(params.TargetFlag), fileName,
		flag(params.FilterFlag), "state=EXCLUDE_NOT_EXPLOITABLE;TO_VERIFY,severity=High;Low,status=new;recurrent",
	)
	defer func() {
		os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
		log.Println("test file removed!")
	}()
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist: "+fileName+printer.FormatJSON)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestResultsShow_ScanIDWithSnoozedAndMutedAllVulnerabilities_NoVulnerabilitiesInScan(t *testing.T) {
	reportFilePath := fmt.Sprintf("%s%s.%s", resultsDirectory, fileName, printer.FormatJSON)

	_ = executeCmdNilAssertion(
		t, "Results show generating JSON report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), ScanIDWithDevAndTestDep,
		flag(params.TargetFormatFlag), printer.FormatJSON,
		flag(params.TargetPathFlag), resultsDirectory,
		flag(params.TargetFlag), fileName,
	)

	defer func() {
		_ = os.RemoveAll(resultsDirectory)
	}()

	assertFileExists(t, reportFilePath)

	var result wrappers.ScanResultsCollection
	readAndUnmarshalFile(t, reportFilePath, &result)

	for _, res := range result.Results {
		assert.Equal(t, "NOT_EXPLOITABLE", res.State, "Should be marked as not exploitable")
	}
}

func TestResultsShow_WithScaHideDevAndTestDependencies_NoVulnerabilitiesInScan(t *testing.T) {
	reportFilePath := fmt.Sprintf("%s%s.%s", resultsDirectory, fileName, printer.FormatJSON)

	_ = executeCmdNilAssertion(
		t, "Results show generating JSON report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), ScanIDWithDevAndTestDep,
		flag(params.TargetFormatFlag), printer.FormatJSON,
		flag(params.TargetPathFlag), resultsDirectory,
		flag(params.TargetFlag), fileName,
		flag(params.ScaHideDevAndTestDepFlag),
	)

	defer func() {
		_ = os.RemoveAll(resultsDirectory)
	}()

	assertFileExists(t, reportFilePath)

	var result wrappers.ScanResultsCollection
	readAndUnmarshalFile(t, reportFilePath, &result)

	assert.Equal(t, len(result.Results), 0, "Should have no results")
}

func assertFileExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	assert.NilError(t, err, "Report file should exist at path "+path)
}

func readAndUnmarshalFile(t *testing.T, path string, v interface{}) {
	file, err := os.ReadFile(path)
	assert.NilError(t, err, "Error reading file at path "+path)

	err = json.Unmarshal(file, v)
	assert.NilError(t, err, "Error unmarshalling JSON data")
}

func TestRiskManagementResults_ReturnResults(t *testing.T) {
	projectName := GenerateRandomProjectNameForScan()
	_ = executeCmdNilAssertion(
		t, "Create project should pass",
		"project", "create",
		flag(params.ProjectName), projectName,
		flag(params.ApplicationName), "ast-phoenix-test", //exist app in deu env.
	)
	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), "data/sources.zip",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.ScanTypes), params.SastType,
	}
	scanId, projectId := executeCreateScan(t, args)

	defer func() {
		_ = executeCmdNilAssertion(
			t, "Delete project should pass",
			"project", "delete",
			flag(params.ProjectIDFlag), projectId,
		)
	}()

	outputBuffer := executeCmdNilAssertion(
		t, "Results risk-management generating JSON report should pass",
		"results", "risk-management",
		flag(params.ProjectIDFlag), projectId,
		flag(params.ScanIDFlag), scanId,
		flag(params.LimitFlag), "20",
	)
	result := wrappers.ASPMResult{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, (len(result.Results) <= 20), "Should have results")
}
