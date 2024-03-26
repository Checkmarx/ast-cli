//go:build integration

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
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

func TestRunGetResultsByScanIdSarifFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sarif")
	// Remove generated sarif file
	removeFileBySuffix(t, printer.FormatSarif)
}
func TestRunGetResultsByScanIdSarifFormatWithContainers(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
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

func TestRunGetResultsByScanIdSonarFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sonar")
	// Remove generated sonar file
	removeFile(t, fileName+"_"+printer.FormatSonar, printer.FormatJSON)
}

func TestRunGetResultsByScanIdSonarFormatWithContainers(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sonar")
	// Remove generated sonar file
	removeFile(t, fileName+"_"+printer.FormatSonar, printer.FormatJSON)
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunGetResultsByScanIdJsonFormatWithContainers(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
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
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
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
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
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
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
}

func TestRunGetResultsByScanIdSummaryMarkdownFormat(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "markdown")
	// Remove generated md file
	removeFileBySuffix(t, "md")
}

func removeFileBySuffix(t *testing.T, suffix string) {
	removeFile(t, fileName, suffix)
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
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
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
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,invalid")
	assert.Equal(t, err.Error(), "report not sent, invalid email address: invalid", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithInvalidOptions(t *testing.T) {
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-options", "invalid")
	assert.Equal(t, err.Error(), "report option \"invalid\" unavailable", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithEmailAndOptions(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,test@test.pt",
		"--report-pdf-options", "Iac-Security,Sast,Sca,ScanSummary")
	assert.NilError(t, err)
}

func TestRunGetResultsGeneratingPdfReporWithOptions(t *testing.T) {
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
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatJSON)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
}

func TestSBOMReportXMLWithContainers(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
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
	removeFile(t, "cx_result.gl-sast-report", "json")
}
func TestRunResultsShow_ContainersFFIsOn_includeContainersResult(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")
	assertContainersPresent(t, true)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}
func TestRunResultsShow_ContainersFFIsOff_excludeContainersResult(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: false}}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")
	assertContainersPresent(t, false)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}
func TestRunResultsShow_AgentIsNotSupported_excludeContainersResult(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: true}}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--agent", "jet-brains")
	assertContainersPresent(t, false)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func assertContainersPresent(t *testing.T, isContainersEnabled bool) {
	bytes, err := os.ReadFile(fileName + "." + printer.FormatJSON)
	assert.NilError(t, err, "Error reading file")
	// Unmarshal the JSON data into the ScanResultsCollection struct
	var scanResultsCollection *wrappers.ScanResultsCollection
	err = json.Unmarshal(bytes, &scanResultsCollection)
	assert.NilError(t, err, "Error unmarshalling JSON data")
	for _, scanResult := range scanResultsCollection.Results {
		if !isContainersEnabled && scanResult.Type == params.ContainersType {
			assert.Assert(t, false, "Containers result should not be present")
		} else if isContainersEnabled && scanResult.Type == params.ContainersType {
			return
		}
	}
	if isContainersEnabled {
		assert.Assert(t, false, "Containers result should be present")
	}
}
func TestRunGetResultsShow_ContainersFFOffAndResultsHasContainersResultsOnly_NilAssertion(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: wrappers.ContainerEngineCLIEnabled, Status: false}}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "CONTAINERS_ONLY", "--report-format", "summaryConsole")
}
