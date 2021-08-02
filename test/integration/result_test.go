// +build integration

package integration

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/checkmarxDev/ast-cli/internal/commands/util"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

// Create a scan and test getting its results
func TestResultListJson(t *testing.T) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	resultCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(resultCommand,
		"result", "list",
		flag(commands.FormatFlag), util.FormatJSON,
		flag(commands.ScanIDFlag), scanID,
	)
	assert.NilError(t, err, "Getting results should pass")

	result := wrappers.ScanResultsCollection{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, len(result.Results) > 0, "Should have results")
}

// Create a scan and test getting its results
func TestResultListSarif(t *testing.T) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	resultCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(resultCommand,
		"result", "list",
		flag(commands.FormatFlag), util.FormatSarif,
		flag(commands.ScanIDFlag), scanID,
	)
	assert.NilError(t, err, "Getting results should pass")

	result := wrappers.ScanResultsCollection{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, len(result.Results) > 0, "Should have results")
}

// Create a scan and test getting its results
func TestResultListPretty(t *testing.T) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	resultCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(resultCommand,
		"result", "list",
		flag(commands.ScanIDFlag), scanID,
	)
	assert.NilError(t, err, "Getting results should pass")

	result, err := io.ReadAll(outputBuffer)
	assert.NilError(t, err, "Reading pretty results should pass")

	assert.Assert(t, len(string(result)) > 0, "Should have results")
	assert.Assert(t, strings.Contains(string(result), "Results"), "Should have results header")
}

// Create a scan and test getting its summary in text
func TestResultSummaryText(t *testing.T) {

	scanID, projectID, result := executeResultSummary(t, util.FormatText)

	assert.Assert(t, strings.Contains(result, "Scan ID: "+scanID))
	assert.Assert(t, strings.Contains(result, "Project ID: "+projectID))
}

// Create a scan and test getting its summary in HTML
func TestResultSummaryHtml(t *testing.T) {

	scanID, projectID, result := executeResultSummary(t, util.FormatHTML)

	assert.Assert(t, strings.Contains(result, fmt.Sprintf("<div>Scan: %s</div>", scanID)))
	assert.Assert(t, strings.Contains(result, projectID))
}

func executeResultSummary(t *testing.T, format string) (string, string, string) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	summaryCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(summaryCommand,
		"result", "summary",
		flag(commands.FormatFlag), format,
		flag(commands.ScanIDFlag), scanID,
	)
	assert.NilError(t, err, "Getting result summary should pass")

	resultBytes, err := io.ReadAll(outputBuffer)
	assert.NilError(t, err, "Reading result summary should pass")

	result := string(resultBytes)
	return scanID, projectID, result
}

// Create a scan and test getting its summary into an html file
func TestResultSummaryOutputFileHtml(t *testing.T) {
	scanID, projectID, html := executeResultSummaryToFile(t, "output.html", util.FormatHTML)

	assert.Assert(t, strings.Contains(html, fmt.Sprintf("<div>Scan: %s</div>", scanID)), "HTML should contain scan id")
	assert.Assert(t, strings.Contains(html, projectID), "HTML should contain project id")
}

// Create a scan and test getting its summary into a text file
func TestResultSummaryOutputFileText(t *testing.T) {
	scanID, projectID, text := executeResultSummaryToFile(t, "output.txt", util.FormatText)

	assert.Assert(t, strings.Contains(text, fmt.Sprintf("Scan ID: %s", scanID)), "Text file should contain scan id")
	assert.Assert(t, strings.Contains(text, fmt.Sprintf("Project ID: %s", projectID)), "Text file should contain project id")
}

func executeResultSummaryToFile(t *testing.T, file string, format string) (string, string, string) {
	scanID, projectID := createScan(t, Dir, Tags)

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	summaryCommand := createASTIntegrationTestCommand(t)

	err := execute(summaryCommand,
		"result", "summary",
		flag(commands.TargetFlag), file,
		flag(commands.ScanIDFlag), scanID,
		flag(commands.FormatFlag), format,
	)
	assert.NilError(t, err, "Getting result summary should pass")

	_, err = os.Stat(file)
	assert.NilError(t, err, "Output file should exist")

	defer func() {
		err := os.Remove(file)
		if err != nil {
			fmt.Println(err)
		}
	}()

	data, err := ioutil.ReadFile(file)
	content := string(data)

	return scanID, projectID, content
}
