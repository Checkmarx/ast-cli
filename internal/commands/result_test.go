//go:build !integration

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/pkg/errors"
	asserts "github.com/stretchr/testify/assert"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gotest.tools/assert"
)

const fileName = "cx_result"

const (
	resultsCommand             = "results"
	codeBashingCommand         = "codebashing"
	vulnerabilityValue         = "Reflected XSS All Clients"
	languageValue              = "PHP"
	cweValue                   = "79"
	jsonValue                  = "json"
	tableValue                 = "table"
	listValue                  = "list"
	secretDetectionLine        = "| Secret Detection          0      1        1      0      0   Completed  |"
	ignorePolicyWarningMessage = "Warning: The --ignore-policy flag was not implemented because you do not have the required permission."
)

func flag(f string) string {
	return "--" + f
}

func TestResultHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "results")
}

func TestResultsExitCode_CompletedScan_PrintCorrectInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{ID: "MOCK", Status: wrappers.ScanCompleted, Engines: []string{params.ScaType, params.SastType, params.KicsType}}
	results := getScannerResponse("", &model)
	assert.Equal(t, len(results), 1, "")
	assert.Equal(t, results[0].ScanID, "MOCK", "")
	assert.Equal(t, results[0].Status, wrappers.ScanCompleted, "")
}

func TestResultsExitCode_OnFailedKicsScanner_PrintCorrectFailedScannerInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-kics-scanner-fail",
		Status: wrappers.ScanFailed,
		StatusDetails: []wrappers.StatusInfo{
			{
				Status:    wrappers.ScanFailed,
				Name:      "kics",
				Details:   "error message from kics scanner",
				ErrorCode: 1234,
			},
			{Status: wrappers.ScanFailed, Name: "general", Details: "timeout", ErrorCode: 1234},
		},
	}

	results := getScannerResponse("", &model)

	assert.Equal(t, len(results), 2, "Scanner results should be empty")
	assert.Equal(t, results[0].Name, "kics", "")
	assert.Equal(t, results[0].ErrorCode, "1234", "")
	assert.Equal(t, results[1].Name, "general", "")
	assert.Equal(t, results[1].ErrorCode, "1234", "")
	assert.Equal(t, results[1].Details, "timeout", "")
}

func TestResultsExitCode_OnFailedKicsAndScaScanners_PrintCorrectFailedScannersInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-multiple-scanner-fails",
		Status: wrappers.ScanFailed,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 2344},
			{Status: wrappers.ScanFailed, Name: "sca", Details: "error message from sca scanner", ErrorCode: 4343},
			{Status: wrappers.ScanFailed, Name: "general", Details: "timeout", ErrorCode: 1234},
		},
	}

	results := getScannerResponse("", &model)

	assert.Equal(t, len(results), 3, "Scanner results should be empty")
	assert.Equal(t, results[0].Name, "kics", "")
	assert.Equal(t, results[0].ErrorCode, "2344", "")
	assert.Equal(t, results[1].Name, "sca", "")
	assert.Equal(t, results[1].ErrorCode, "4343", "")
	assert.Equal(t, results[2].Name, "general", "")
	assert.Equal(t, results[2].ErrorCode, "1234", "")
	assert.Equal(t, results[2].Details, "timeout", "")
}

func TestResultsExitCode_OnRequestedFailedScanner_PrintCorrectFailedScannerInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-multiple-scanner-fails",
		Status: wrappers.ScanFailed,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 2344},
			{Status: wrappers.ScanFailed, Name: "sca", Details: "error message from sca scanner", ErrorCode: 4343},
			{Status: wrappers.ScanFailed, Name: "general", Details: "timeout", ErrorCode: 1234},
		},
	}

	results := getScannerResponse("sca", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].Name, "sca", "")
	assert.Equal(t, results[0].ErrorCode, "4343", "")
}

func TestResultsExitCode_OnPartialScan_PrintOnlyFailedScannersInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-sca-fail-partial-id",
		Status: wrappers.ScanPartial,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanCompleted, Name: "sast"},
			{Status: wrappers.ScanCanceled, Name: "sca", Details: "error message from sca scanner", ErrorCode: 4343},
			{Status: wrappers.ScanCompleted, Name: "general"},
		},
	}

	results := getScannerResponse("", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].ScanID, "fake-scan-id-sca-fail-partial-id", "")
	assert.Equal(t, results[0].Status, "Partial", "")
}

func runScanCommand(t *testing.T, agent, scanID string) *wrappers.ScanResultsCollection {
	clearFlags()

	_, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", scanID, "--report-format", "json", "--agent", agent)
	assert.NilError(t, err)

	file, err := os.Open(fileName + ".json")
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer func() {
		file.Close()
		os.Remove(fileName + ".json")
	}()

	fileContents, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var results wrappers.ScanResultsCollection
	err = json.Unmarshal(fileContents, &results)
	assert.NilError(t, err)
	return &results
}

func TestRunScsResultsShow_ASTCLI_AgentShouldShowAllResults(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "SCS_ONLY", "--report-format", "json", "--agent", params.DefaultAgent)
	assertTypePresentJSON(t, params.SCSScorecardType, 1)
	assertTypePresentJSON(t, params.SCSSecretDetectionType, 2)
	assertTotalCountJSON(t, 3)

	removeFileBySuffix(t, printer.FormatJSON)
	mock.SetScsMockVarsToDefault()
}

func TestRunScsResultsShow_VSCode_AgentShouldNotShowScorecardResults(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "SCS_ONLY", "--report-format", "json", "--agent", params.VSCodeAgent)
	assertTypePresentJSON(t, params.SCSScorecardType, 0)
	assertTypePresentJSON(t, params.SCSSecretDetectionType, 2)
	assertTotalCountJSON(t, 2)

	removeFileBySuffix(t, printer.FormatJSON)
	mock.SetScsMockVarsToDefault()
}

func TestRunScsResultsShow_Jetbrains_AgentShouldShowScsResults(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "SCS_ONLY", "--report-format", "json", "--agent", params.JetbrainsAgent)
	assertTypePresentJSON(t, params.SCSScorecardType, 0)
	assertTypePresentJSON(t, params.SCSSecretDetectionType, 2)
	assertTotalCountJSON(t, 2)

	removeFileBySuffix(t, printer.FormatJSON)
	mock.SetScsMockVarsToDefault()
}

func TestRunWithoutScsResults_Other_AgentsShouldNotShowScsResults(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "SAST_ONLY", "--report-format", "json", "--agent", params.EclipseAgent)
	assertTypePresentJSON(t, params.SCSScorecardType, 0)
	assertTypePresentJSON(t, params.SCSSecretDetectionType, 0)
	assertTotalCountJSON(t, 1)

	removeFileBySuffix(t, printer.FormatJSON)
	mock.SetScsMockVarsToDefault()
}

func TestRunNilResults_Other_AgentsShouldNotShowAnyResults(t *testing.T) {
	clearFlags()

	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK_NO_VULNERABILITIES", "--report-format", "json", "--agent", params.VisualStudioAgent)
	assertTypePresentJSON(t, params.SCSScorecardType, 0)
	assertTypePresentJSON(t, params.SCSSecretDetectionType, 0)
	assertTotalCountJSON(t, 0)

	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunScsResultsShow_Other_AgentShouldShowSCSResults(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "SCS_ONLY", "--report-format", "json", "--agent", params.VisualStudioAgent)
	assertTypePresentJSON(t, params.SCSScorecardType, 0)
	assertTypePresentJSON(t, params.SCSSecretDetectionType, 2)
	assertTotalCountJSON(t, 2)

	removeFileBySuffix(t, printer.FormatJSON)
	mock.SetScsMockVarsToDefault()
}

func TestFilterScsResultsByAgent_ShouldIncludeSCSAndSAST(t *testing.T) {
	results := &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{Type: params.SCSScorecardType},
			{Type: params.ScsType},
			{Type: params.SastType},
		},
	}

	filteredResults := filterScsResultsByAgent(results, params.VisualStudioAgent)

	hasSCS := false
	hasSCSScorecard := false
	hasSAST := false

	for _, result := range filteredResults.Results {
		switch result.Type {
		case params.ScsType:
			hasSCS = true
		case params.SCSScorecardType:
			hasSCSScorecard = true
		case params.SastType:
			hasSAST = true
		}
	}

	assert.Assert(t, hasSCS, "Expected SCS type to be included for Visual Studio agent")
	assert.Assert(t, !hasSCSScorecard, "Expected SCSScorecard type to be excluded for Visual Studio agent")
	assert.Assert(t, hasSAST, "Expected SAST type to be excluded for Visual Studio agent")
	assert.Equal(t, len(filteredResults.Results), 2, "Expected 2 results (SCS and SAST) after filtering for Visual Studio agent")
}

func TestResultsExitCode_OnCanceledScan_PrintOnlyScanIDAndStatusCanceledToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-kics-fail-sast-canceled-id",
		Status: wrappers.ScanCanceled,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanCompleted, Name: "general"},
			{Status: wrappers.ScanCompleted, Name: "sast"},
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 6455},
		},
	}

	results := getScannerResponse("", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].ScanID, "fake-scan-id-kics-fail-sast-canceled-id", "")
	assert.Equal(t, results[0].Status, wrappers.ScanCanceled, "")
}

func TestResultsExitCode_OnCanceledScanWithRequestedSuccessfulScanner_PrintOnlyScanIDAndStatusCanceledToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-kics-fail-sast-canceled-id",
		Status: wrappers.ScanCanceled,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanCompleted, Name: "general"},
			{Status: wrappers.ScanCompleted, Name: "sast"},
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 6455},
		},
	}

	results := getScannerResponse("sast", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].ScanID, "fake-scan-id-kics-fail-sast-canceled-id", "")
	assert.Equal(t, results[0].Status, wrappers.ScanCanceled, "")
}

func TestResultsExitCode_OnCanceledScanWithRequestedFailedScanner_PrintOnlyScanIDAndStatusCanceledToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-kics-fail-sast-canceled-id",
		Status: wrappers.ScanCanceled,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanCompleted, Name: "general"},
			{Status: wrappers.ScanCompleted, Name: "sast"},
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 6455},
		},
	}

	results := getScannerResponse("kics", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].ScanID, "fake-scan-id-kics-fail-sast-canceled-id", "")
	assert.Equal(t, results[0].Status, wrappers.ScanCanceled, "")
}

func TestResultsExitCode_NoScanIdSent_FailCommandWithError(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "exit-code")
	assert.Equal(t, err.Error(), errorConstants.ScanIDRequired, "Wrong expected error message")
}

func TestResultsExitCode_OnErrorScan_FailCommandWithError(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "exit-code", "--scan-id", "fake-error-id")
	assert.Equal(t, err.Error(), "Failed showing a scan: fake error message", "Wrong expected error message")
}

func TestRunGetResultsByScanIdSarifFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sarif")
	// Remove generated sarif file
	removeFileBySuffix(t, printer.FormatSarif)
}
func TestRunGetResultsByScanIdSarifFormatWithContainers(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sarif")
	// Remove generated sarif file
	removeFileBySuffix(t, printer.FormatSarif)
}

func TestParseSarifEmptyResultSast(t *testing.T) {
	emptyResult := &wrappers.ScanResult{}
	result := parseSarifResultSast(emptyResult, nil)
	if result != nil {
		t.Errorf("Expected nil result for empty ScanResultData.Nodes, got %v", result)
	}
}

func TestParseSarifResultSastClampsZeroColumns(t *testing.T) {
	result := &wrappers.ScanResult{
		ScanResultData: wrappers.ScanResultData{
			Nodes: []*wrappers.ScanResultNode{
				{FileName: "/src/app.go", Line: 10, Column: 0, Length: 0},
				{FileName: "/src/app.go", Line: 12, Column: 0, Length: 5},
			},
		},
	}

	sarifResults := parseSarifResultSast(result, nil)
	assert.Assert(t, len(sarifResults) == 1)
	assert.Assert(t, len(sarifResults[0].Locations) == 1)

	primary := sarifResults[0].Locations[0].PhysicalLocation.Region
	assert.Equal(t, uint(1), primary.StartColumn)
	assert.Equal(t, uint(2), primary.EndColumn)

	threadLocations := sarifResults[0].CodeFlows[0].ThreadFlows[0].Locations
	assert.Equal(t, uint(1), threadLocations[0].Location.PhysicalLocation.Region.StartColumn)
	assert.Equal(t, uint(2), threadLocations[0].Location.PhysicalLocation.Region.EndColumn)
	assert.Equal(t, uint(1), threadLocations[1].Location.PhysicalLocation.Region.StartColumn)
	assert.Equal(t, uint(6), threadLocations[1].Location.PhysicalLocation.Region.EndColumn)
}

func TestParseSarifResultKicsClampsZeroStartLine(t *testing.T) {
	result := &wrappers.ScanResult{
		ScanResultData: wrappers.ScanResultData{
			Filename: "/Dockerfile",
			Line:     0,
		},
	}

	sarifResults := parseSarifResultKics(result, nil)
	assert.Assert(t, len(sarifResults) == 1)
	assert.Equal(t, uint(1), sarifResults[0].Locations[0].PhysicalLocation.Region.StartLine)
}

func TestParseSarifResultsSscsClampsZeroStartLine(t *testing.T) {
	result := &wrappers.ScanResult{
		Type:        params.SCSSecretDetectionType,
		Description: "secret found",
		ScanResultData: wrappers.ScanResultData{
			Filename: "config.yaml",
			Line:     0,
		},
	}

	sarifResults := parseSarifResultsSscs(result, nil)
	assert.Assert(t, len(sarifResults) == 1)
	assert.Equal(t, uint(1), sarifResults[0].Locations[0].PhysicalLocation.Region.StartLine)
}

func TestRunGetResultsByScanIdSonarFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sonar")
	// Remove generated sonar file
	removeFile(t, fileName+"_"+printer.FormatSonar, printer.FormatJSON)
}

func TestRunGetResultsByScanIdSonarFormatWithContainers(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sonar")
	// Remove generated sonar file
	removeFile(t, fileName+"_"+printer.FormatSonar, printer.FormatJSON)
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestDecodeHTMLEntitiesInResults(t *testing.T) {
	// Setup: Creating test data with HTML entities
	results := createTestScanResultsCollection()

	decodeHTMLEntitiesInResults(results)

	expectedFullName := `SomeClass<T>`
	expectedName := `Name with "quotes"`

	if results.Results[0].ScanResultData.Nodes[0].FullName != expectedFullName {
		t.Errorf("expected FullName to be %q, got %q", expectedFullName, results.Results[0].ScanResultData.Nodes[0].FullName)
	}

	if results.Results[0].ScanResultData.Nodes[0].Name != expectedName {
		t.Errorf("expected Name to be %q, got %q", expectedName, results.Results[0].ScanResultData.Nodes[0].Name)
	}
}

func TestRunGetResultsByScanIdJsonFormatWithContainers(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunGetResultsByScanIdJsonFormatWithSastRedundancy(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--sast-redundancy")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunGetResultsByScanIdSummaryJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryJSON")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunGetResultsByScanIdSummaryJsonFormatWithContainers(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryJSON")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunGetResultsByScanIdSummaryHtmlFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryHTML")

	// Remove generated html file
	removeFileBySuffix(t, printer.FormatHTML)
}

func TestRunGetResultsByScanIdSummaryHtmlFormatWithContainers(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryHTML")

	// Remove generated html file
	removeFileBySuffix(t, printer.FormatHTML)
}

func TestRunGetResultsByScanIdSummaryConsoleFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
}

func TestRunGetResultsByScanIdSummaryMarkdownFormatWithContainers(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "markdown")
	// Remove generated md file
	removeFileBySuffix(t, "md")
}

func TestRunGetResultsByScanIdSummaryConsoleFormatWithContainers(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
}

func TestRunGetResultsByScanIdSummaryMarkdownFormat(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "markdown")
	// Remove generated md file
	removeFileBySuffix(t, "md")
}

func createTestScanResultsCollection() *wrappers.ScanResultsCollection {
	return &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{
				Description:     "Vulnerability in SomeComponent",
				DescriptionHTML: "Description with quotes",
				ScanResultData: wrappers.ScanResultData{
					Nodes: []*wrappers.ScanResultNode{
						{
							FullName: "SomeClass&lt;T&gt;",
							Name:     "Name with &quot;quotes&quot;",
						},
					},
				},
			},
		},
	}
}

func removeFileBySuffix(t *testing.T, suffix string) {
	switch suffix {
	case printer.FormatSonar:
		removeFile(t, fileName+sonarTypeLabel, printer.FormatJSON)
	default:
		removeFile(t, fileName, suffix)
	}
}

func removeFile(t *testing.T, prefix, suffix string) {
	err := os.Remove(fmt.Sprintf("%s.%s", prefix, suffix))
	assert.NilError(t, err, "Error removing file, check if report file created")
}

func TestRunGetResultsByScanIdPDFFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "pdf")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatPDF)
	// Remove generated pdf file
	removeFileBySuffix(t, printer.FormatPDF)
}

func TestRunGetResultsByScanIdPDFFormatWithContainers(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "pdf")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatPDF)
	// Remove generated pdf file
	removeFileBySuffix(t, printer.FormatPDF)
}

func TestRunGetResultsByScanIdWrongFormat(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "invalidFormat")
	assert.Equal(t, err.Error(), "bad report format invalidFormat", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithWrongFilterFormat(t *testing.T) {
	_ = execCmdNotNilAssertion(
		t,
		"results",
		"show",
		"--scan-id",
		"MOCK",
		"--report-format",
		"sarif",
		"--filter",
		"limit40",
	)
}

func TestRunGetResultsByScanIdWithMissingOrEmptyScanId(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "show")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")

	err = execCmdNotNilAssertion(t, "results", "show", "--scan-id", "")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithEmptyOutputPath(t *testing.T) {
	_ = execCmdNotNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--output-path", "")
}

func TestRunGetCodeBashingWithoutLanguage(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue)
	assert.Equal(t, err.Error(), "required flag(s) \"language\" not set", "Wrong expected error message")
}

func TestRunGetCodeBashingWithoutVulnerabilityType(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.LanguageFlag),
		languageValue)
	assert.Equal(t, err.Error(), "required flag(s) \"vulnerability-type\" not set", "Wrong expected error message")
}

func TestRunGetCodeBashingWithoutCweId(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue,
		flag(params.LanguageFlag),
		languageValue)
	assert.Equal(t, err.Error(), "required flag(s) \"cwe-id\" not set", "Wrong expected error message")
}

func TestRunGetCodeBashingWithFormatJson(t *testing.T) {
	execCmdNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue,
		flag(params.LanguageFlag),
		languageValue,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.FormatFlag),
		jsonValue)
}

func TestRunGetCodeBashingWithFormatTable(t *testing.T) {
	execCmdNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue,
		flag(params.LanguageFlag),
		languageValue,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.FormatFlag),
		tableValue)
}

func TestRunGetCodeBashingWithFormatList(t *testing.T) {
	execCmdNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue,
		flag(params.LanguageFlag),
		languageValue,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.FormatFlag),
		listValue)
}

func TestResultBflHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "results bfl")
}

func TestRunGetBflWithMissingOrEmptyScanIdAndQueryId(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "bfl")
	assert.Equal(t, err.Error(), "required flag(s) \"query-id\", \"scan-id\" not set")

	err = execCmdNotNilAssertion(t, "results", "bfl", "--scan-id", "")
	assert.Equal(t, err.Error(), "required flag(s) \"query-id\" not set")

	err = execCmdNotNilAssertion(t, "results", "bfl", "--query-id", "")
	assert.Equal(t, err.Error(), "required flag(s) \"scan-id\" not set")
}

func TestRunGetBflWithMultipleScanIdsAndQueryIds(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "bfl", "--scan-id", "MOCK1,MOCK2", "--query-id", "MOCK")
	assert.Equal(t, err.Error(), "Multiple scan-ids are not allowed.")

	err = execCmdNotNilAssertion(t, "results", "bfl", "--scan-id", "MOCK1", "--query-id", "MOCK1,MOCK2")
	assert.Equal(t, err.Error(), "Multiple query-ids are not allowed.")
}

func TestRunGetBFLByScanIdAndQueryId(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "results", "bfl", "--scan-id", "MOCK", "--query-id", "MOCK")
	assert.NilError(t, err)
}

func TestRunGetBFLByScanIdAndQueryIdWithFormatJson(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "results", "bfl", "--scan-id", "MOCK", "--query-id", "MOCK", "--format", "JSON")
	assert.NilError(t, err)
}

func TestRunGetBFLByScanIdAndQueryIdWithFormatList(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "results", "bfl", "--scan-id", "MOCK", "--query-id", "MOCK", "--format", "List")
	assert.NilError(t, err)
}

func TestRunGetResultsGeneratingPdfReportWithInvalidEmail(t *testing.T) {
	clearFlags()
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,invalid")
	assert.Equal(t, err.Error(), "report not sent, invalid email address: invalid", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithInvalidOptions(t *testing.T) {
	clearFlags()
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-options", "invalid")
	assert.Equal(t, err.Error(), "report option \"invalid\" unavailable", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithInvalidImprovedOptions(t *testing.T) {
	clearFlags()
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-options", "scan-information")
	assert.Equal(t, err.Error(), "report option \"scan-information\" unavailable", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithEmailAndOptions(t *testing.T) {
	clearFlags()
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,test@test.pt",
		"--report-pdf-options", "Iac-Security,Sast,Sca,ScanSummary")
	assert.NilError(t, err)
}

func TestRunGetResultsGeneratingPdfReportWithOptionsImprovedMappingHappens(t *testing.T) {
	clearFlags()
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,test@test.pt",
		"--report-pdf-options", "Iac-Security,Sast,Sca,scansummary,scanresults")
	assert.NilError(t, err)
}

func TestRunGetResultsGeneratingPdfReportWithInvalidOptionsImproved(t *testing.T) {
	clearFlags()
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,test@test.pt",
		"--report-pdf-options", "Iac-Security,Sast,Sca,scan-information")
	assert.Error(t, err, "report option \"scan-information\" unavailable")
}

func TestRunGetResultsGeneratingPdfReportWithOptions(t *testing.T) {
	clearFlags()
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--output-name", fileName,
		"--report-pdf-options", "Iac-Security,Sast,Sca,ScanSummary")
	defer func() {
		removeFileBySuffix(t, printer.FormatPDF)
		fmt.Println("test file removed!")
	}()
	assert.NilError(t, err)
	_, err = os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "report file should exist: "+fileName+printer.FormatPDF)
}

func TestRunGetResultsGeneratingJsonV2Report(t *testing.T) {
	clearFlags()
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "json-v2",
		"--scan-id", "MOCK",
		"--output-name", fileName)
	defer func() {
		removeFileBySuffix(t, printer.FormatJSON)
		fmt.Println("test file removed!")
	}()
	assert.NilError(t, err)
	_, err = os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
	assert.NilError(t, err, "report file should exist: "+fileName+printer.FormatJSON)
}

func TestSBOMReportInvalidSBOMOption(t *testing.T) {
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "sbom",
		"--scan-id", "MOCK",
		"--report-sbom-format", "invalid")
	assert.Equal(t, err.Error(), "invalid SBOM option: invalid", "Wrong expected error message")
}

func TestSBOMReportJson(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatJSON)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
}

func TestSBOMReportXML(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom", "--report-sbom-format", "CycloneDxXml")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatXML)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
}

func TestSBOMReportJsonWithContainers(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatJSON)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
}

func TestSBOMReportXMLWithContainers(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom", "--report-sbom-format", "CycloneDxXml")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatXML)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
}

func TestRunGetResultsByScanIdGLFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "gl-sast")
	// Run test for gl-sast report type
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatGLSast))
}
func TestRunResultsShow_ContainersFFIsOn_includeContainersResult(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")
	assertTypePresentJSON(t, params.ContainersType, 1)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunResultsShow_jetbrainsIsNotSupported_excludeContainersResult(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--agent", "jetbrains")
	assertTypePresentJSON(t, params.ContainersType, 0)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunResultsShow_EclipseIsNotSupported_excludeContainersResult(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--agent", "Eclipse")
	assertTypePresentJSON(t, params.ContainersType, 0)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunResultsShow_VsCodeIsNotSupported_excludeContainersResult(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--agent", "vs code")
	assertTypePresentJSON(t, params.ContainersType, 0)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunResultsShow_VisualStudioIsNotSupported_excludeContainersResult(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--agent", "Visual Studio")
	assertTypePresentJSON(t, params.ContainersType, 0)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func assertTypePresentJSON(t *testing.T, resultType string, expectedResultTypeCount int) {
	reportBytes, err := os.ReadFile(fileName + "." + printer.FormatJSON)
	assert.NilError(t, err, "Error reading file")
	// Unmarshal the JSON data into the ScanResultsCollection struct
	var scanResultsCollection *wrappers.ScanResultsCollection
	err = json.Unmarshal(reportBytes, &scanResultsCollection)
	assert.NilError(t, err, "Error unmarshalling JSON data")
	actualResultTypeCount := 0
	for i := range scanResultsCollection.Results {
		scanResult := scanResultsCollection.Results[i]
		if scanResult.Type == resultType {
			actualResultTypeCount++
		}
	}
	assert.Equal(t, actualResultTypeCount, expectedResultTypeCount,
		fmt.Sprintf("Expected %s result count to be %d, but found %d results", resultType, expectedResultTypeCount, actualResultTypeCount))
}

func assertTotalCountJSON(t *testing.T, expectedResultTypeCount uint) {
	reportBytes, err := os.ReadFile(fileName + "." + printer.FormatJSON)
	assert.NilError(t, err, "Error reading file")
	// Unmarshal the JSON data into the ScanResultsCollection struct
	var scanResultsCollection *wrappers.ScanResultsCollection
	err = json.Unmarshal(reportBytes, &scanResultsCollection)
	assert.NilError(t, err, "Error unmarshalling JSON data")

	assert.Equal(t, scanResultsCollection.TotalCount, expectedResultTypeCount,
		fmt.Sprintf("Expected total count to be %d, but actual total count is %d", expectedResultTypeCount, scanResultsCollection.TotalCount))
}

func assertTypePresentSonar(t *testing.T, resultType string, expectedResultTypeCount int) {
	reportBytes, err := os.ReadFile(fileName + sonarTypeLabel + "." + printer.FormatJSON)
	assert.NilError(t, err, "Error reading file")
	// Unmarshal the JSON data into the ScanResultsCollection struct
	var scanResultsCollection *wrappers.ScanResultsSonar
	err = json.Unmarshal(reportBytes, &scanResultsCollection)
	assert.NilError(t, err, "Error unmarshalling JSON data")

	actualResultTypeCount := 0
	for _, rule := range scanResultsCollection.Rules {
		if rule.EngineID == resultType {
			for _, issue := range scanResultsCollection.Issues {
				if issue.RuleID == rule.ID {
					actualResultTypeCount++
				}
			}
		}
	}
	assert.Equal(t, actualResultTypeCount, expectedResultTypeCount,
		fmt.Sprintf("Expected %s result count to be %d, but found %d results", resultType, expectedResultTypeCount, actualResultTypeCount))
}

func assertTypePresentSarif(t *testing.T, resultType string, expectedResultTypeCount int) {
	reportBytes, err := os.ReadFile(fileName + "." + printer.FormatSarif)
	assert.NilError(t, err, "Error reading file")
	// Unmarshal the JSON data into the ScanResultsCollection struct
	var scanResultsCollection *wrappers.SarifResultsCollection
	err = json.Unmarshal(reportBytes, &scanResultsCollection)
	assert.NilError(t, err, "Error unmarshalling SARIF data")
	resultTypeRuleSuffix := fmt.Sprintf("(%s)", resultType)
	actualResultTypeCount := 0
	for i := range scanResultsCollection.Runs[0].Results {
		scanResult := scanResultsCollection.Runs[0].Results[i]
		if strings.HasSuffix(scanResult.RuleID, resultTypeRuleSuffix) {
			actualResultTypeCount++
			assertRulePresentSarif(t, scanResult.RuleID, scanResultsCollection)
		}
	}
	assert.Equal(t, actualResultTypeCount, expectedResultTypeCount,
		fmt.Sprintf("Expected %s result count to be %d, but found %d results", resultType, expectedResultTypeCount, actualResultTypeCount))
}

func assertURINonEmpty(t *testing.T) {
	reportBytes, err := os.ReadFile(fileName + "." + printer.FormatSarif)
	assert.NilError(t, err, "Error reading SARIF file")
	var scanResults *wrappers.SarifResultsCollection
	err = json.Unmarshal(reportBytes, &scanResults)
	assert.NilError(t, err, "Error unmarshalling SARIF results")

	for i := range scanResults.Runs[0].Results {
		locations := scanResults.Runs[0].Results[i].Locations
		if len(locations) > 0 && strings.Contains(locations[0].PhysicalLocation.ArtifactLocation.URI, "This alert has no associated file") {
			return
		}
	}
	assert.Assert(t, false, "expected a SARIF result with the no-associated-file placeholder URI, found none")
}

func assertRulePresentSarif(t *testing.T, ruleID string, scanResultsCollection *wrappers.SarifResultsCollection) {
	for i := range scanResultsCollection.Runs[0].Tool.Driver.Rules {
		rule := scanResultsCollection.Runs[0].Tool.Driver.Rules[i]
		if rule.ID == ruleID {
			return
		}
	}
	assert.Assert(t, false, fmt.Sprintf("RuleID %s found in SARIF result not found in rules of SARIF report", ruleID))
}

func assertResultsPresentSummaryJSON(t *testing.T, isResultsEnabled bool, scanType string, numberOfIssues *int) {
	reportBytes, err := os.ReadFile(fileName + "." + printer.FormatJSON)
	assert.NilError(t, err, "Error reading file")
	// Unmarshal the JSON data into the ScanResultsCollection struct
	var scanResultSummary *wrappers.ResultSummary
	err = json.Unmarshal(reportBytes, &scanResultSummary)
	assert.NilError(t, err, "Error unmarshalling JSON data")

	// Test presence of Issues field
	scanTypeCapitalized := cases.Title(language.Und).String(scanType)
	IssuesFieldName := scanTypeCapitalized + "Issues"
	reflectedScanResultSummary := reflect.ValueOf(scanResultSummary).Elem()
	IssuesField := reflectedScanResultSummary.FieldByName(IssuesFieldName)

	assert.Equal(t, IssuesField.IsValid(), true, fmt.Sprintf("field %s not found in ResultSummary struct definition", IssuesFieldName))
	assert.Equal(t, !IssuesField.IsNil(), isResultsEnabled, fmt.Sprintf("Expected field %s to be present: %t", IssuesFieldName, isResultsEnabled))

	if !IssuesField.IsNil() && numberOfIssues != nil {
		assert.Equal(t, *IssuesField.Interface().(*int), *numberOfIssues, fmt.Sprintf("Expected field %s to have value: %d", IssuesFieldName, *numberOfIssues))
	}

	// Test presence of Scs Overview field
	if scanType == params.ScsType {
		ScsOverviewField := reflectedScanResultSummary.FieldByName("SCSOverview")
		assert.Equal(t, ScsOverviewField.IsValid(), true, fmt.Sprintf("field %s not found in ResultSummary struct definition ", ScsOverviewField))
		assert.Equal(t, !ScsOverviewField.IsNil(), isResultsEnabled, fmt.Sprintf("Expected field %s to be present: %t", ScsOverviewField, isResultsEnabled))
	}

	for engine := range scanResultSummary.EnginesResult {
		if !isResultsEnabled && engine == scanType {
			assert.Assert(t, false, fmt.Sprintf("%s result summary should not be present", scanType))
		} else if isResultsEnabled && engine == scanType {
			return
		}
	}
	if isResultsEnabled {
		assert.Assert(t, false, "%s result summary should be present", scanType)
	}
}

func TestRunGetResultsByScanIdGLSastAndAScaFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "gl-sast,gl-sca")
	// Run test for gl-sast report type
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatGLSast))
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatGLSca))
}

func TestRunGetResultsByScanIdGLScaFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "gl-sca")
	// Run test for gl-sca report type
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatGLSca))
}

func Test_addPackageInformation(t *testing.T) {
	var dependencyPath = wrappers.DependencyPath{ID: "test-1"}
	var dependencyArray = [][]wrappers.DependencyPath{{dependencyPath}}
	resultsModel := &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{
				Type: "sca", // Assuming this matches commonParams.ScaType
				ScanResultData: wrappers.ScanResultData{
					PackageIdentifier: "pkg-123",
				},
				ID: "CVE-2021-23-424",
				VulnerabilityDetails: wrappers.VulnerabilityDetails{
					CvssScore: 5.0,
					CveName:   "cwe-789",
				},
			},
		},
	}
	scaPackageModel := &[]wrappers.ScaPackageCollection{
		{
			ID:                  "pkg-123",
			FixLink:             "",
			DependencyPathArray: dependencyArray,
		},
	}
	scaTypeModel := &[]wrappers.ScaTypeCollection{
		{}}

	resultsModel = addPackageInformation(resultsModel, scaPackageModel, scaTypeModel)

	expectedFixLink := "https://devhub.checkmarx.com/cve-details/CVE-2021-23-424"
	actualFixLink := resultsModel.Results[0].ScanResultData.ScaPackageCollection.FixLink
	assert.Equal(t, expectedFixLink, actualFixLink, "FixLink should match the result ID")
}

func TestRunGetResultsByScanIdGLSastFormat_NoVulnerabilities_Success(t *testing.T) {
	// Execute the command and perform nil assertion
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK_NO_VULNERABILITIES", "--report-format", "gl-sast")

	// Run test for gl-sast report type
	// Check if the file exists and vulnerabilities is empty, then delete the file
	if _, err := os.Stat(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSast)); err == nil {
		t.Logf("File exists: %s.%s", fileName, printer.FormatGLSast)
		resultsData, err := os.ReadFile(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSast))
		if err != nil {
			t.Logf("Failed to read file: %v", err)
		}

		var results wrappers.GlSastResultsCollection
		if err := json.Unmarshal(resultsData, &results); err != nil {
			t.Logf("Failed to unmarshal JSON: %v", err)
		}
		assert.Equal(t, len(results.Vulnerabilities), 0, "No vulnerabilities should be found")
		if err := os.Remove(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSast)); err != nil {
			t.Logf("Failed to delete file: %v", err)
		}
		t.Log("File deleted successfully.")
	}
}

func TestRunGetResultsByScanIdGLScaFormat_NoVulnerabilities_Success(t *testing.T) {
	// Execute the command and perform nil assertion
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK_NO_VULNERABILITIES", "--report-format", "gl-sca")

	// Run test for gl-sca report type
	// Check if the file exists and vulnerabilities is empty, then delete the file
	if _, err := os.Stat(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSca)); err == nil {
		t.Logf("File exists: %s.%s", fileName, printer.FormatGLSca)
		resultsData, err := os.ReadFile(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSca))
		if err != nil {
			t.Logf("Failed to read file: %v", err)
		}

		var results wrappers.GlScaResultsCollection
		if err := json.Unmarshal(resultsData, &results); err != nil {
			t.Logf("Failed to unmarshal JSON: %v", err)
		}
		assert.Equal(t, len(results.Vulnerabilities), 0, "No vulnerabilities should be found")
		if err := os.Remove(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSca)); err != nil {
			t.Logf("Failed to delete file: %v", err)
		}
		t.Log("File deleted successfully.")
	}
}

func TestRunGetResultsByScanIdSummaryConsoleFormat_ScsNotScanned_ScsMissingInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = false
	mock.ScsScanPartial = false
	mock.ScorecardScanned = false

	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	fmt.Print(stdoutString)

	scsSummary := "| SCS               -       -        -       -      -       -       |"
	assert.Equal(t, strings.Contains(stdoutString, scsSummary), true,
		"Expected SCS summary:"+scsSummary)
	secretDetectionSummary := "Secret Detection"
	assert.Equal(t, !strings.Contains(stdoutString, secretDetectionSummary), true,
		"Expected Secret Detection summary to be missing:"+secretDetectionSummary)
	scorecardSummary := "Scorecard"
	assert.Equal(t, !strings.Contains(stdoutString, scorecardSummary), true,
		"Expected Scorecard summary to be missing:"+scorecardSummary)

	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSummaryConsoleFormat_ScsCompleted_ScsCompletedInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.CVSSV3Enabled, Status: true}

	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	ansiRegexp := regexp.MustCompile("\x1b\\[[0-9;]*[mK]")
	cleanString := ansiRegexp.ReplaceAllString(stdoutString, "")
	fmt.Print(stdoutString)

	TotalResults := "Total Results: 11"
	assert.Equal(t, strings.Contains(cleanString, TotalResults), true,
		"Expected: "+TotalResults)
	TotalSummary := "| TOTAL             0       6        3       2      0   Completed   |"
	assert.Equal(t, strings.Contains(cleanString, TotalSummary), true,
		"Expected TOTAL summary: "+TotalSummary)
	scsSummary := "| SCS               0       1        1       1      0   Completed   |"
	assert.Equal(t, strings.Contains(cleanString, scsSummary), true,
		"Expected SCS summary:"+scsSummary)
	secretDetectionSummary := secretDetectionLine
	assert.Equal(t, strings.Contains(cleanString, secretDetectionSummary), true,
		"Expected Secret Detection summary:"+secretDetectionLine)
	scorecardSummary := "| Scorecard                 0      0        0      1      0   Completed  |"
	assert.Equal(t, strings.Contains(cleanString, scorecardSummary), true,
		"Expected Scorecard summary:"+scorecardSummary)

	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSummaryConsoleFormat_ScsPartial_ScsPartialInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = true
	mock.ScorecardScanned = true
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.CVSSV3Enabled, Status: true}

	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	ansiRegexp := regexp.MustCompile("\x1b\\[[0-9;]*[mK]")
	cleanString := ansiRegexp.ReplaceAllString(stdoutString, "")
	fmt.Print(stdoutString)

	TotalResults := "Total Results: 10"
	assert.Equal(t, strings.Contains(cleanString, TotalResults), true,
		"Expected: "+TotalResults)
	TotalSummary := "| TOTAL             0       6        3       1      0   Completed   |"
	assert.Equal(t, strings.Contains(cleanString, TotalSummary), true,
		"Expected TOTAL summary: "+TotalSummary)
	scsSummary := "| SCS               0       1        1       0      0   Partial     |"
	assert.Equal(t, strings.Contains(cleanString, scsSummary), true,
		"Expected SCS summary:"+scsSummary)
	secretDetectionSummary := secretDetectionLine
	assert.Equal(t, strings.Contains(cleanString, secretDetectionSummary), true,
		"Expected Secret Detection summary:"+secretDetectionLine)
	scorecardSummary := " | Scorecard                 0      0        0      0      0   Failed     |"
	assert.Equal(t, strings.Contains(cleanString, scorecardSummary), true,
		"Expected Scorecard summary:"+scorecardSummary)

	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSummaryConsoleFormat_ScsScorecardNotScanned_ScorecardMissingInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = false
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.CVSSV3Enabled, Status: true}

	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	fmt.Print(stdoutString)

	scsSummary := "| SCS               0       1        1       0      0   Completed   |"
	assert.Equal(t, strings.Contains(stdoutString, scsSummary), true,
		"Expected SCS summary:"+scsSummary)
	secretDetectionSummary := secretDetectionLine
	assert.Equal(t, strings.Contains(stdoutString, secretDetectionSummary), true,
		"Expected Secret Detection summary:"+secretDetectionLine)
	scorecardSummary := "| Scorecard                 -      -        -      -      -       -      |"
	assert.Equal(t, strings.Contains(stdoutString, scorecardSummary), true,
		"Expected Scorecard summary:"+scorecardSummary)

	mock.SetScsMockVarsToDefault()
}

func TestGetResultsSummaryConsoleFormatWithCriticalDisabled(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.CVSSV3Enabled, Status: false}
	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	fmt.Print(stdoutString)

	totalSummary := "| TOTAL           N/A       5        2       1      0   Completed   |"
	assert.Equal(t, strings.Contains(stdoutString, totalSummary), true,
		"Expected Total summary without critical:"+totalSummary)

	mock.SetScsMockVarsToDefault()
}

func Test_enhanceWithScanSummary(t *testing.T) {
	tests := []struct {
		name                string
		summary             *wrappers.ResultSummary
		results             *wrappers.ScanResultsCollection
		featureFlagsWrapper wrappers.FeatureFlagsWrapper
		expectedIssues      int
	}{
		{
			name:    "scan summary with no vulnerabilities",
			summary: createEmptyResultSummary(),
			results: &wrappers.ScanResultsCollection{
				Results:    []*wrappers.ScanResult{},
				TotalCount: 0,
				ScanID:     "MOCK",
			},
			featureFlagsWrapper: mock.FeatureFlagsMockWrapper{},
			expectedIssues:      0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			enhanceWithScanSummary(tt.summary, tt.results, tt.featureFlagsWrapper)
			assert.Equal(t, tt.expectedIssues, tt.summary.TotalIssues)
		})
	}
}

func createEmptyResultSummary() *wrappers.ResultSummary {
	var containersIssues = new(int)
	*containersIssues = 0

	return &wrappers.ResultSummary{
		TotalIssues:      0,
		CriticalIssues:   0,
		HighIssues:       0,
		MediumIssues:     0,
		LowIssues:        0,
		InfoIssues:       0,
		SastIssues:       0,
		ScaIssues:        0,
		KicsIssues:       0,
		ContainersIssues: containersIssues,
		SCSOverview:      &wrappers.SCSOverview{},
		APISecurity: wrappers.APISecFilteredResult{
			APICount:        0,
			TotalRisksCount: 0,
			SeverityCount:   map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0},
		},
		EnginesEnabled: []string{"sast", "sca", "kics", "containers"},
		EnginesResult: wrappers.EnginesResultsSummary{
			params.SastType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
				Info:     0,
			},
			params.ScaType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
				Info:     0,
			},
			params.KicsType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
				Info:     0,
			},
			params.APISecType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
			},
			params.ContainersType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
			},
		},
	}
}
func TestPrintPoliciesSummary_WhenNoRolViolated_ShouldNotContainPolicyViolation(t *testing.T) {
	summary := &wrappers.ResultSummary{
		Policies: &wrappers.PolicyResponseModel{
			Status: "Success",
			Policies: []wrappers.Policy{
				{
					RulesViolated: []string{},
				},
			},
			BreakBuild: false,
		},
	}
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printPoliciesSummary(summary, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err) // Handle the error if io.Copy fails
	}
	output := buf.String()
	assert.Assert(t, !strings.Contains(output, "Policy Management Violation "), "Output should not contain 'Policy Management Violation'")
}

func TestRunGetResultsByScanIdJSONFormat_SCSFlagEnabled_SCSPresentInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")
	assertTypePresentJSON(t, params.SCSScorecardType, 1)
	assertTypePresentJSON(t, params.SCSSecretDetectionType, 2)

	removeFileBySuffix(t, printer.FormatJSON)
	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSonarFormat_SCSFlagEnabled_SCSPresentInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sonar")
	assertTypePresentSonar(t, params.SCSScorecardType, 1)
	assertTypePresentSonar(t, params.SCSSecretDetectionType, 2)

	removeFileBySuffix(t, printer.FormatSonar)
	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSarifFormat_SCSFlagEnabled_SCSNonEmpty_URI_PresentInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sarif")
	assertTypePresentSarif(t, params.SCSScorecardType, 1)
	assertTypePresentSarif(t, params.SCSSecretDetectionType, 2)
	assertURINonEmpty(t)
	removeFileBySuffix(t, printer.FormatSarif)
	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSummaryJSONFormat_SCSFlagEnabled_SCSPresentInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true
	ScsFlagValue := true
	expectedScsIssues := 3

	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryJSON")

	assertResultsPresentSummaryJSON(t, ScsFlagValue, params.ScsType, &expectedScsIssues)

	removeFileBySuffix(t, printer.FormatJSON)
	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSummaryMarkdownFormat_SCSFlagEnabled_SCSPresentInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "markdown")
	// Read the contents of the file
	markdownBytes, err := os.ReadFile(fmt.Sprintf("%s.%s", fileName, "md"))
	assert.NilError(t, err, "Error reading file")

	markdownString := string(markdownBytes)
	assert.Equal(t, strings.Contains(markdownString, "SCS"), true, "SCS should be present in the markdown file")

	// Remove generated md file
	removeFileBySuffix(t, "md")
	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSummaryHtmlFormat_SCSFlagEnabled_SCSPresentInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true

	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryHTML")
	// Read the contents of the file
	htmlBytes, err := os.ReadFile(fmt.Sprintf("%s.%s", fileName, "html"))
	assert.NilError(t, err, "Error reading file")

	htmlString := string(htmlBytes)
	assert.Equal(t, strings.Contains(htmlString, "SCS"), true, "SCS should be present in the html file")

	// Remove generated html file
	removeFileBySuffix(t, "html")
	mock.SetScsMockVarsToDefault()
}

func TestFilterScsResultsByAgent_ShouldExcludeSCSAndContainers(t *testing.T) {
	results := &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{Type: params.SCSScorecardType},
			{Type: params.ScsType},
			{Type: params.ContainersType},
			{Type: params.SastType},
		},
	}

	filteredResults := filterScsResultsByAgent(results, params.JetbrainsAgent)

	hasSCSScorecard := false
	hasSCS := false
	hasContainers := false
	hasSAST := false

	for _, result := range filteredResults.Results {
		switch result.Type {
		case params.SCSScorecardType:
			hasSCSScorecard = true
		case params.ScsType:
			hasSCS = true
		case params.ContainersType:
			hasContainers = true
		case params.SastType:
			hasSAST = true
		}
	}

	assert.Assert(t, !hasSCSScorecard, "Expected SCSScorecard type to be excluded for Jetbrains agent")
	assert.Assert(t, hasSCS, "Expected SCS type to be included in Jetbrains agent results")
	assert.Assert(t, hasContainers, "Expected Containers type to be included in Jetbrains agent results")
	assert.Assert(t, hasSAST, "Expected SAST type to be included in Jetbrains agent results")
	assert.Equal(t, len(filteredResults.Results), 3, "Expected only 3 results after filtering for Jetbrains agent")
}

func TestRiskManagementHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "results", "risk-management")
}

func TestRiskManagement_ShouldFFBeFalseAndReturnError(t *testing.T) {
	clearFlags()
	err := execCmdNotNilAssertion(t, "results", "risk-management")
	assert.Equal(t, err.Error(), "Risk management results are currently unavailable for your tenant.", "Expected error message")

}

func TestRiskManagement(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.RiskManagementEnabled, Status: true}
	execCmdNilAssertion(t, "results", "risk-management")
}

func Test_addPackageInformation_DependencyTypes(t *testing.T) {
	// Create dependency paths with different types
	var dependencyPaths = [][]wrappers.DependencyPath{
		{{
			ID:            "dev-pkg",
			IsDevelopment: true,
		}},
		{{
			ID:            "test-pkg",
			IsDevelopment: false,
		}},
	}

	// Create results model with two results - one dev and one test
	resultsModel := &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{
				Type: "sca",
				ScanResultData: wrappers.ScanResultData{
					PackageIdentifier: "dev-pkg",
				},
			},
			{
				Type: "sca",
				ScanResultData: wrappers.ScanResultData{
					PackageIdentifier: "test-pkg",
				},
			},
		},
	}

	// Create package model with different dev/test settings
	scaPackageModel := &[]wrappers.ScaPackageCollection{
		{
			ID:                      "dev-pkg",
			DependencyPathArray:     dependencyPaths[:1],
			IsDevelopmentDependency: true,
			IsTestDependency:        false,
		},
		{
			ID:                      "test-pkg",
			DependencyPathArray:     dependencyPaths[1:],
			IsDevelopmentDependency: false,
			IsTestDependency:        true,
		},
	}

	scaTypeModel := &[]wrappers.ScaTypeCollection{{}}

	// Execute the function
	resultsModel = addPackageInformation(resultsModel, scaPackageModel, scaTypeModel)

	// Get the results
	devPackage := resultsModel.Results[0].ScanResultData.ScaPackageCollection
	testPackage := resultsModel.Results[1].ScanResultData.ScaPackageCollection

	// Verify the fields were transferred correctly
	assert.Equal(t, true, devPackage.IsDevelopmentDependency, "First package should be marked as development dependency")
	assert.Equal(t, false, devPackage.IsTestDependency, "First package should not be marked as test dependency")

	assert.Equal(t, false, testPackage.IsDevelopmentDependency, "Second package should not be marked as development dependency")
	assert.Equal(t, true, testPackage.IsTestDependency, "Second package should be marked as test dependency")
}

func TestIgnorePolicyWithNoPermission(t *testing.T) {
	policyResponseModel := wrappers.PolicyResponseModel{}
	policyResponseModel.BreakBuild = false

	policy := wrappers.Policy{}
	policy.Name = "MOCK_NAME1"
	policy.RulesViolated = make([]string, 1)
	policy.BreakBuild = true
	policy.Description = "MOCK_DESC1"
	policy.Tags = make([]string, 0)

	var policies []wrappers.Policy
	policies = append(policies, policy)
	policyResponseModel.Policies = policies
	summary := &wrappers.ResultSummary{
		Policies: &policyResponseModel,
	}
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printPoliciesSummary(summary, true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err) // Handle the error if io.Copy fails
	}
	output := buf.String()
	assert.Assert(t, strings.Contains(output, ignorePolicyWarningMessage), "’Ignore Policy flag omitted because you dont have permission’ should not be present in the output")
}

func TestIgnorePolicyWithPermission(t *testing.T) {
	policyResponseModel := wrappers.PolicyResponseModel{}
	policyResponseModel.BreakBuild = false

	policy := wrappers.Policy{}
	policy.Name = "MOCK_NAME2"
	policy.RulesViolated = make([]string, 1)
	policy.BreakBuild = true
	policy.Description = "MOCK_DESC2"
	policy.Tags = make([]string, 0)

	var policies []wrappers.Policy
	policies = append(policies, policy)
	policyResponseModel.Policies = policies
	summary := &wrappers.ResultSummary{
		Policies: &policyResponseModel,
	}
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printPoliciesSummary(summary, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err) // Handle the error if io.Copy fails
	}
	output := buf.String()
	assert.Assert(t, !strings.Contains(output, ignorePolicyWarningMessage), "’Ignore Policy flag omitted because you dont have permission’ should not be present in the output")
}

func TestParseGlSastVulnerability_QueryDescriptionLink_Succeed(t *testing.T) {
	mockResult := createMockScanResult("q1234", "c5678")
	glSast := &wrappers.GlSastResultsCollection{}
	summary := &wrappers.ResultSummary{
		BaseURI:   "https://example.com/overview",
		ScanID:    "scanID",
		ProjectID: "projectID",
	}
	expectedURL := "https://example.com/results/scanID/projectID/sast/description/c5678/q1234"

	glSast = parseGlSastVulnerability(mockResult, glSast, summary)

	assert.Assert(t, len(glSast.Vulnerabilities) > 0)

	actualURL := extractURLFromDescription(glSast.Vulnerabilities[0].Description)

	assert.Equal(t, actualURL, expectedURL, "QueryDescriptionLink URL does not match expected format")
}

func TestParseGlSastVulnerability_QueryDescriptionLink_Negative(t *testing.T) {
	mockResult := createMockScanResult("", "")
	glSast := &wrappers.GlSastResultsCollection{}
	summary := &wrappers.ResultSummary{
		BaseURI:   "invalid-url",
		ScanID:    "scanID",
		ProjectID: "projectID",
	}
	expectedPattern := "/results/scanID/projectID/sast/description//"

	glSast = parseGlSastVulnerability(mockResult, glSast, summary)

	assert.Assert(t, len(glSast.Vulnerabilities) > 0)
	vuln := glSast.Vulnerabilities[0]

	assert.Assert(t, strings.Contains(vuln.Description, expectedPattern),
		"URL should contain pattern with empty values")

	actualURL := extractURLFromDescription(vuln.Description)
	assert.Assert(t, actualURL != "", "Extracted URL should not be empty")
}

func createMockScanResult(queryID, cweID string) *wrappers.ScanResult {
	return &wrappers.ScanResult{
		Type: "sast",
		ScanResultData: wrappers.ScanResultData{
			QueryName: "TestQuery",
			QueryID:   queryID,
			Nodes: []*wrappers.ScanResultNode{
				{
					FileName: "file.go",
					Line:     42,
					Length:   1,
				},
			},
		},
		VulnerabilityDetails: wrappers.VulnerabilityDetails{
			CweID: cweID,
		},
		ID:          "vuln-1",
		Description: "desc-",
		Severity:    "high",
	}
}

func extractURLFromDescription(description string) string {
	parts := strings.Split(description, "http")
	if len(parts) == 1 {
		return "http" + strings.Split(parts[0], " ")[0]
	} else if len(parts) > 1 {
		return "http" + strings.Split(parts[1], " ")[0]
	}
	return ""
}

func TestGetFilterResultsForAPISecScanner(t *testing.T) {
	mockWrapper := mock.RisksOverviewMockWrapper{}
	scanID := "test-scan-id"
	resultsParams := map[string]string{}

	result, err := getFilterResultsForAPISecScanner(mockWrapper, scanID, resultsParams)
	assert.NilError(t, err)
	assert.Assert(t, result == nil, "Expected nil result for empty entries")

	mockEntries := []wrappers.APISecRiskEntry{
		{Severity: "Critical", Origin: "code", State: "to_verify"},
		{Severity: "High", Origin: "documentation", State: "to_verify"},
		{Severity: "Medium", Origin: "code", State: "to_verify"},
		{Severity: "Low", Origin: "documentation", State: "to_verify"},
		{Severity: "Critical", Origin: "code", State: "not_exploitable"},
	}
	mockWrapperWithEntries := &mock.RisksOverviewMockWrapperWithEntries{Entries: mockEntries}
	result, err = getFilterResultsForAPISecScanner(mockWrapperWithEntries, scanID, resultsParams)
	assert.NilError(t, err)
	assert.Assert(t, result != nil, "Expected non-nil result for entries")
	if result == nil {
		return
	}
	assert.Equal(t, result.SeverityCount["critical"], 1)
	assert.Equal(t, result.SeverityCount["high"], 1)
	assert.Equal(t, result.SeverityCount["medium"], 1)
	assert.Equal(t, result.SeverityCount["low"], 1)
	assert.Equal(t, result.TotalRisksCount, 4)
	assert.Equal(t, len(result.RiskDistribution), 2)
	for _, dist := range result.RiskDistribution {
		if dist.Origin == "code" {
			assert.Equal(t, dist.Total, 2)
		}
		if dist.Origin == "documentation" {
			assert.Equal(t, dist.Total, 2)
		}
	}
}

func TestGetAISCInfoFromScanSummary_Success(t *testing.T) {
	mockWrapper := &mock.ScanSummaryMockWrapper{}
	scanID := "test-scan-id"

	result, err := getAISCInfoFromScanSummary(mockWrapper, scanID)

	assert.NilError(t, err)
	assert.Assert(t, result != nil, "Expected non-nil result")
	if result == nil {
		return
	}
	assert.Equal(t, result.TotalAssets, 0, "Expected TotalAssets to be 0")
	assert.Equal(t, result.TotalAssetTypes, 0, "Expected TotalAssetTypes to be 0")
}

func TestGetAISCInfoFromScanSummary_WithNonZeroValues(t *testing.T) {
	// Create a custom mock wrapper with non-zero values
	mockWrapper := &customScanSummaryMockWrapper{
		assetsCounter:     10,
		assetTypesCounter: 5,
	}
	scanID := "test-scan-id"

	result, err := getAISCInfoFromScanSummary(mockWrapper, scanID)

	assert.NilError(t, err)
	assert.Assert(t, result != nil, "Expected non-nil result")
	if result == nil {
		return
	}
	assert.Equal(t, result.TotalAssets, 10, "Expected TotalAssets to be 10")
	assert.Equal(t, result.TotalAssetTypes, 5, "Expected TotalAssetTypes to be 5")
}

func TestGetAISCInfoFromScanSummary_EmptyScansSummaries(t *testing.T) {
	mockWrapper := &customScanSummaryMockWrapper{
		emptySummaries: true,
	}
	scanID := "test-scan-id"

	result, err := getAISCInfoFromScanSummary(mockWrapper, scanID)

	assert.NilError(t, err)
	assert.Assert(t, result == nil, "Expected nil result for empty summaries")
}

func TestGetAISCInfoFromScanSummary_NilScanSummaryModel(t *testing.T) {
	mockWrapper := &customScanSummaryMockWrapper{
		nilModel: true,
	}
	scanID := "test-scan-id"

	result, err := getAISCInfoFromScanSummary(mockWrapper, scanID)

	assert.NilError(t, err)
	assert.Assert(t, result == nil, "Expected nil result for nil scan summary model")
}

func TestGetAISCInfoFromScanSummary_Error(t *testing.T) {
	mockWrapper := &customScanSummaryMockWrapper{
		returnError: true,
	}
	scanID := "test-scan-id"

	result, err := getAISCInfoFromScanSummary(mockWrapper, scanID)

	assert.Assert(t, err != nil, "Expected error")
	assert.Assert(t, result == nil, "Expected nil result on error")
	if err == nil {
		return
	}
	assert.Assert(t, strings.Contains(err.Error(), "AISC"), "Expected error message to contain 'AISC'")
	assert.Assert(t, strings.Contains(err.Error(), failedListingResults), "Expected error message to contain failedListingResults")
}

func TestGetAISCInfoFromScanSummary_WebError(t *testing.T) {
	mockWrapper := &customScanSummaryMockWrapper{
		returnWebError: true,
	}
	scanID := "test-scan-id"

	result, err := getAISCInfoFromScanSummary(mockWrapper, scanID)

	assert.Assert(t, err != nil, "Expected error")
	assert.Assert(t, result == nil, "Expected nil result on web error")
	if err == nil {
		return
	}
	assert.Assert(t, strings.Contains(err.Error(), "AISC"), "Expected error message to contain 'AISC'")
	assert.Assert(t, strings.Contains(err.Error(), failedListingResults), "Expected error message to contain failedListingResults")
	assert.Assert(t, strings.Contains(err.Error(), "CODE: 400"), "Expected error message to contain error code")
	assert.Assert(t, strings.Contains(err.Error(), "Bad Request"), "Expected error message to contain error message")
}

// Custom mock wrapper for testing different scenarios
type customScanSummaryMockWrapper struct {
	assetsCounter     int
	assetTypesCounter int
	emptySummaries    bool
	nilModel          bool
	returnError       bool
	returnWebError    bool
}

func (m *customScanSummaryMockWrapper) GetScanSummaryByScanID(scanID string) (*wrappers.ScanSummariesModel, *wrappers.WebError, error) {
	if m.returnError {
		return nil, nil, errors.New("mock error from GetScanSummaryByScanID")
	}
	if m.returnWebError {
		return nil, &wrappers.WebError{
			Code:    400,
			Message: "Bad Request",
		}, nil
	}
	if m.nilModel {
		return nil, nil, nil
	}
	if m.emptySummaries {
		return &wrappers.ScanSummariesModel{
			ScansSummaries: []wrappers.ScanSumaries{},
			TotalCount:     0,
		}, nil, nil
	}
	return &wrappers.ScanSummariesModel{
		ScansSummaries: []wrappers.ScanSumaries{
			{
				ScanID: scanID,
				AiscCounters: wrappers.AiscCounters{
					AssetsCounter:     m.assetsCounter,
					AssetTypesCounter: m.assetTypesCounter,
				},
			},
		},
		TotalCount: 1,
	}, nil, nil
}

// writeSourceFile creates a file with the given content inside dir and returns its path relative to dir.
func writeSourceFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	asserts.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	asserts.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return filepath.ToSlash(name)
}

// assertValidSonarRange asserts a textRange satisfies the bounds SonarScanner enforces on import.
func assertValidSonarRange(t *testing.T, tr *wrappers.SonarTextRange, totalLines, lineLength uint) {
	t.Helper()
	if tr == nil {
		return // omitted entirely, so the issue is file level and always valid
	}
	asserts.NotZero(t, tr.StartLine, "startLine must be set when textRange is present")
	asserts.LessOrEqual(t, tr.StartLine, totalLines, "startLine must exist in the file")

	if tr.StartColumn == 0 && tr.EndColumn == 0 {
		return // line level range, always accepted
	}
	asserts.LessOrEqual(t, tr.StartColumn, lineLength, "startColumn must not exceed line length")
	asserts.LessOrEqual(t, tr.EndColumn, lineLength, "endColumn must not exceed line length")
	asserts.Less(t, tr.StartColumn, tr.EndColumn, "range must move forward")
}

func TestClampSonarColumns(t *testing.T) {
	tests := []struct {
		name        string
		startColumn uint
		length      uint
		lineLength  uint
		wantStart   uint
		wantEnd     uint
		wantEmit    bool
	}{
		{
			// Reported case: a 12 character line with Column=12, Length=6 produced endColumn 17.
			name:        "overflowing end column is clamped to line length",
			startColumn: 11, length: 6, lineLength: 12,
			wantStart: 11, wantEnd: 12, wantEmit: true,
		},
		{
			name:        "valid range is preserved unchanged",
			startColumn: 4, length: 5, lineLength: 40,
			wantStart: 4, wantEnd: 9, wantEmit: true,
		},
		{
			name:        "range ending exactly at line length is valid",
			startColumn: 0, length: 12, lineLength: 12,
			wantStart: 0, wantEnd: 12, wantEmit: true,
		},
		{
			name:        "zero length node falls back to the whole line",
			startColumn: 4, length: 0, lineLength: 40,
			wantStart: 0, wantEnd: 40, wantEmit: true,
		},
		{
			name:        "start beyond end of line falls back to the whole line",
			startColumn: 50, length: 3, lineLength: 12,
			wantStart: 0, wantEnd: 12, wantEmit: true,
		},
		{
			name:        "empty line yields no column range",
			startColumn: 0, length: 3, lineLength: 0,
			wantStart: 0, wantEnd: 0, wantEmit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, emit := clampSonarColumns(tt.startColumn, tt.length, tt.lineLength)
			asserts.Equal(t, tt.wantEmit, emit)
			asserts.Equal(t, tt.wantStart, start)
			asserts.Equal(t, tt.wantEnd, end)
			if emit {
				asserts.LessOrEqual(t, end, tt.lineLength)
				asserts.Less(t, start, end)
			}
		})
	}
}

func TestSonarLineIndexLineLength(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	lfFile := writeSourceFile(t, dir, "lf.ts", "one\nthree3\n")
	crlfFile := writeSourceFile(t, dir, "crlf.ts", "one\r\nthree3\r\n")
	bomFile := writeSourceFile(t, dir, "bom.ts", string(byteOrderMarkRune)+"abc\n")
	utf8File := writeSourceFile(t, dir, "utf8.ts", "héllo→\n")
	emptyLineFile := writeSourceFile(t, dir, "nested/dir/empty.ts", "\nabc\n")

	index := newSonarLineIndex()

	t.Run("counts characters on an LF file", func(t *testing.T) {
		length, status := index.resolveLine(lfFile, 2)
		asserts.Equal(t, lineStatusOK, status)
		asserts.Equal(t, uint(6), length)
	})

	t.Run("CRLF terminator is not counted", func(t *testing.T) {
		length, status := index.resolveLine(crlfFile, 2)
		asserts.Equal(t, lineStatusOK, status)
		asserts.Equal(t, uint(6), length)
	})

	t.Run("leading BOM is not counted", func(t *testing.T) {
		length, status := index.resolveLine(bomFile, 1)
		asserts.Equal(t, lineStatusOK, status)
		asserts.Equal(t, uint(3), length)
	})

	t.Run("multi byte characters count as one character each", func(t *testing.T) {
		length, status := index.resolveLine(utf8File, 1)
		asserts.Equal(t, lineStatusOK, status)
		// "héllo→" is 6 characters but 9 bytes.
		asserts.Equal(t, uint(6), length)
	})

	t.Run("empty line reports zero length", func(t *testing.T) {
		length, status := index.resolveLine(emptyLineFile, 1)
		asserts.Equal(t, lineStatusOK, status)
		asserts.Equal(t, uint(0), length)
	})

	t.Run("leading slash in the report path is tolerated", func(t *testing.T) {
		length, status := index.resolveLine("/"+lfFile, 1)
		asserts.Equal(t, lineStatusOK, status)
		asserts.Equal(t, uint(3), length)
	})

	t.Run("line past end of a readable file is reported missing", func(t *testing.T) {
		_, status := index.resolveLine(lfFile, 99)
		asserts.Equal(t, lineStatusLineMissing, status)
	})

	t.Run("line zero of a readable file is reported missing", func(t *testing.T) {
		_, status := index.resolveLine(lfFile, 0)
		asserts.Equal(t, lineStatusLineMissing, status)
	})

	t.Run("missing file is reported unknown, not missing line", func(t *testing.T) {
		_, status := index.resolveLine("does/not/exist.ts", 1)
		asserts.Equal(t, lineStatusFileUnknown, status)
	})

	t.Run("empty file name is reported unknown", func(t *testing.T) {
		_, status := index.resolveLine("", 1)
		asserts.Equal(t, lineStatusFileUnknown, status)
	})
}

func TestSonarLineIndexRejectsPathsOutsideBaseDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	index := newSonarLineIndex()

	for _, fileName := range []string{
		"../escape.ts",
		"../../../../etc/passwd",
		"nested/../../escape.ts",
	} {
		t.Run("rejects "+fileName, func(t *testing.T) {
			_, ok := index.resolveSourcePath(fileName)
			asserts.False(t, ok, "path outside the working directory must be rejected")
		})
	}

	t.Run("rejects an absolute path", func(t *testing.T) {
		absolute := filepath.Join(dir, "abs.ts")
		asserts.NoError(t, os.WriteFile(absolute, []byte("abc\n"), 0o600))
		// Absolute inputs are refused even when they resolve inside baseDir.
		_, ok := index.resolveSourcePath(absolute)
		asserts.False(t, ok)
	})

	t.Run("accepts a plain relative path inside the base directory", func(t *testing.T) {
		name := writeSourceFile(t, dir, "inside.ts", "abc\n")
		resolved, ok := index.resolveSourcePath(name)
		asserts.True(t, ok)
		asserts.True(t, strings.HasSuffix(resolved, "inside.ts"))
	})
}

// TestParseSonarTextRangeCustomerRegression reproduces the reported "17 is not a valid line offset" failure end to end.
func TestParseSonarTextRangeCustomerRegression(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	const componentLine = "@Component({" // 12 characters, line 10 below
	source := "import { Component, inject, OnInit } from '@angular/core'\n" +
		"import { DomSanitizer } from '@angular/platform-browser'\n" +
		"import jwtDecode from 'jwt-decode'\n" +
		"import { TranslateModule } from '@ngx-translate/core'\n" +
		"import { MatCardModule } from '@angular/material/card'\n" +
		"\n\n\n\n" + // lines 6 to 9
		componentLine + "\n" + // line 10
		"  selector: 'app-last-login-ip',\n"

	fileName := writeSourceFile(t,
		dir,
		"cxone-sq-integration/juice-shop-master/frontend/src/app/last-login-ip/last-login-ip.component.ts",
		source,
	)

	index := newSonarLineIndex()
	node := &wrappers.ScanResultNode{
		FileName: fileName,
		Line:     10,
		Column:   12,
		Length:   6,
	}

	textRange := parseSonarTextRange(node, index)

	asserts.NotNil(t, textRange)
	asserts.Equal(t, uint(10), textRange.StartLine)
	asserts.Equal(t, uint(11), textRange.StartColumn)
	asserts.Equal(t, uint(12), textRange.EndColumn, "endColumn must be clamped to the 12 character line, not 17")
	assertValidSonarRange(t, textRange, 11, uint(len(componentLine)))
}

func TestParseSonarTextRangeFallsBackToLineLevel(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	index := newSonarLineIndex()

	t.Run("unreadable file keeps the line and emits no columns", func(t *testing.T) {
		node := &wrappers.ScanResultNode{FileName: "absent.ts", Line: 7, Column: 12, Length: 6}
		textRange := parseSonarTextRange(node, index)
		asserts.NotNil(t, textRange)
		asserts.Equal(t, uint(7), textRange.StartLine)
		asserts.Zero(t, textRange.StartColumn, "columns are omitempty, so zero drops them from the report")
		asserts.Zero(t, textRange.EndColumn)
	})

	t.Run("empty line emits no columns", func(t *testing.T) {
		fileName := writeSourceFile(t, dir, "blank.ts", "\nabc\n")
		node := &wrappers.ScanResultNode{FileName: fileName, Line: 1, Column: 3, Length: 4}
		textRange := parseSonarTextRange(node, index)
		asserts.NotNil(t, textRange)
		asserts.Equal(t, uint(1), textRange.StartLine)
		asserts.Zero(t, textRange.StartColumn)
		asserts.Zero(t, textRange.EndColumn)
	})
}

// TestParseSonarTextRangeOmitsRangeForMissingLine covers the minified-asset case where the engine reports a line past the end of the file.
func TestParseSonarTextRangeOmitsRangeForMissingLine(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fileName := writeSourceFile(t, dir, "assets/private/dat.gui.min.js", "var a=1\nvar b=2\n")
	index := newSonarLineIndex()

	node := &wrappers.ScanResultNode{FileName: fileName, Line: 803, Column: 5, Length: 4}
	textRange := parseSonarTextRange(node, index)

	asserts.Nil(t, textRange, "textRange must be omitted when the line does not exist in the file")
}

// TestParseSonarAllLocationsAreValid asserts every location of a result with overflowing nodes satisfies SonarQube's invariant.
func TestParseSonarAllLocationsAreValid(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	const line = "@Component({" // 12 characters
	fileName := writeSourceFile(t, dir, "app/widget.component.ts", line+"\n"+line+"\n")

	results := &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{
				Type: params.SastType,
				ScanResultData: wrappers.ScanResultData{
					QueryName: "Angular_Client_Stored_DOM_XSS",
					Nodes: []*wrappers.ScanResultNode{
						{FileName: fileName, Line: 1, Column: 12, Length: 6},
						{FileName: fileName, Line: 2, Column: 40, Length: 9},
						{FileName: fileName, Line: 1, Column: 1, Length: 0},
						{FileName: "missing.ts", Line: 3, Column: 5, Length: 4},
						// Line beyond end of a readable file: must be dropped.
						{FileName: fileName, Line: 803, Column: 5, Length: 4},
					},
				},
			},
		},
	}

	issues, _ := parseSonar(results)

	asserts.Len(t, issues, 1)
	// The node on line 803 is dropped: it has no valid textRange, which is mandatory on secondary locations as well.
	asserts.Len(t, issues[0].SecondaryLocations, 3)

	all := append([]wrappers.SonarLocation{issues[0].PrimaryLocation}, issues[0].SecondaryLocations...)
	for i := range all {
		location := all[i]
		t.Run("location "+string(rune('A'+i)), func(t *testing.T) {
			if location.FilePath != fileName {
				// A file that is not on disk keeps its line but drops the columns.
				asserts.NotNil(t, location.TextRange)
				asserts.NotZero(t, location.TextRange.StartLine)
				asserts.Zero(t, location.TextRange.StartColumn)
				asserts.Zero(t, location.TextRange.EndColumn)
				return
			}
			assertValidSonarRange(t, location.TextRange, 2, uint(len(line)))
		})
	}

	// SonarQube rejects the whole report if any secondary location has a nil textRange.
	for _, location := range issues[0].SecondaryLocations {
		asserts.NotNil(t, location.TextRange, "secondary locations must always carry a textRange")
	}
}
