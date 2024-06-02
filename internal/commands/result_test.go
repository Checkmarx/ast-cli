//go:build !integration

package commands

import (
	"fmt"
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

const fileName = "cx_result"

const (
	resultsCommand     = "results"
	codeBashingCommand = "codebashing"
	vulnerabilityValue = "Reflected XSS All Clients"
	languageValue      = "PHP"
	cweValue           = "79"
	jsonValue          = "json"
	tableValue         = "table"
	listValue          = "list"
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
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatSarif))
}

func TestParseSarifEmptyResultSast(t *testing.T) {
	emptyResult := &wrappers.ScanResult{}
	result := parseSarifResultSast(emptyResult, nil)
	if result != nil {
		t.Errorf("Expected nil result for empty ScanResultData.Nodes, got %v", result)
	}
}

func TestRunGetResultsByScanIdSonarFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sonar")

	// Remove generated sonar file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatSonar))
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")

	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
}

func TestRunGetResultsByScanIdJsonFormatWithSastRedundancy(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--sast-redundancy")

	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
}

func TestRunGetResultsByScanIdSummaryJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryJSON")

	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
}

func TestRunGetResultsByScanIdSummaryHtmlFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryHTML")

	// Remove generated html file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatHTML))
}

func TestRunGetResultsByScanIdSummaryConsoleFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
}

func TestRunGetResultsByScanIdPDFFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "pdf")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatPDF)
	// Remove generated pdf file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
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
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.NewScanReportEnabled, Status: false}}
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,invalid")
	assert.Equal(t, err.Error(), "report not sent, invalid email address: invalid", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithInvalidOptions(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.NewScanReportEnabled, Status: false}}
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-options", "invalid")
	assert.Equal(t, err.Error(), "report option \"invalid\" unavailable", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithInvalidImprovedOptions(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.NewScanReportEnabled, Status: false}}
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-options", "scan-information")
	assert.Equal(t, err.Error(), "report option \"scan-information\" unavailable", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithEmailAndOptions(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.NewScanReportEnabled, Status: false}}
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
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.NewScanReportEnabled, Status: true}}
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
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.NewScanReportEnabled, Status: true}}
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
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.NewScanReportEnabled, Status: false}}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--output-name", fileName,
		"--report-pdf-options", "Iac-Security,Sast,Sca,ScanSummary")
	defer func() {
		os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
		fmt.Println("test file removed!")
	}()
	assert.NilError(t, err)
	_, err = os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "report file should exist: "+fileName+printer.FormatPDF)
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

func TestSBOMReportXMLWithProxy(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom", "--report-sbom-format", "CycloneDxXml", "--report-sbom-local-flow")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatXML)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
}

func TestRunGetResultsByScanIdGLFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "gl-sast")
	// Run test for gl-sast report type
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatGL))
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
