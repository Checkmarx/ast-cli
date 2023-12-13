//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

var projectID string

func TestSastUpdateAndGetPredicatesForSimilarityId(t *testing.T) {

	fmt.Println("Step 1: Testing the command 'triage update' to update an issue from the project.")

	scanID, projectID := getRootScan(t)
	_ = executeCmdNilAssertion(
		t, "Results show generating JSON report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID, flag(params.TargetFormatFlag), printer.FormatJSON,
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

	//similarityID := "1ddba286f9d7caa81c1a605df93351df959c5abc4a576a8cdcc1a047b656afb9"

	similarityID := result.Results[0].SimilarityID
	//if strings.HasPrefix(result.Results[0].SimilarityID, "-") {
	//	similarityID = result.Results[0].SimilarityID[1:]
	//}
	state := "Confirmed"
	if result.Results[0].State != "Urgent" {
		state = "Urgent"
	}
	severity := "High"
	if result.Results[0].Severity != "Medium" {
		severity = "Medium"
	}
	comment := "Testing CLI Command for triage."
	scanType := result.Results[0].Type

	//state := "Urgent"
	//severity := "Medium"
	//comment := "Testing CLI Command for triage."
	//scanType := "kics"

	args := []string{
		"triage", "update",
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.StateFlag), state,
		flag(params.SeverityFlag), severity,
		flag(params.CommentFlag), comment,
		flag(params.ScanTypeFlag), scanType,
	}

	_, outputBufferForStep1 := executeCommand(t, args...)
	_, readingError := io.ReadAll(outputBufferForStep1)
	assert.NilError(t, readingError, "Reading result should pass")

	err, _ = executeCommand(t, args...)
	assert.NilError(t, err, "Updating the predicate with same values should pass.")

	fmt.Println("Step 2: Testing the command 'triage show' to get the same predicate back.")
	outputBufferForStep2 := executeCmdNilAssertion(
		t, "Predicates should be fetched.", "triage", "show",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.ScanTypeFlag), scanType,
	)

	predicateResult := []wrappers.Predicate{}
	fmt.Println(outputBufferForStep2)
	_ = unmarshall(t, outputBufferForStep2, &predicateResult, "Reading results should pass")

	assert.Assert(t, (len(predicateResult)) == 1, "Should have 1 predicate as the result.")

	deleteScanAndProject()

}

func TestGetAndUpdatePredicateWithInvalidScannerType(t *testing.T) {

	err, _ := executeCommand(
		t, "triage", "update",
		flag(params.ProjectIDFlag), "1234",
		flag(params.SimilarityIDFlag), "1234",
		flag(params.StateFlag), "URGENT",
		flag(params.SeverityFlag), "High",
		flag(params.CommentFlag), "",
		flag(params.ScanTypeFlag), "NoScanner",
	)

	fmt.Println(err)
	assertError(t, err, "Invalid scan type NoScanner")

	_, responseBuffer := executeCommand(
		t, "triage", "show",
		flag(params.ProjectIDFlag), "1234",
		flag(params.SimilarityIDFlag), "1234",
		flag(params.ScanTypeFlag), "sca",
	)

	scaPredicate := wrappers.PredicatesCollectionResponseModel{}
	_ = unmarshall(t, responseBuffer, &scaPredicate, "Reading predicate should pass")
	assert.Assert(t, scaPredicate.TotalCount == 0, "Predicate with scanner-type sca should have 0 as the result.")
}

func TestPredicateWithInvalidValues(t *testing.T) {

	err, _ := executeCommand(
		t, "triage", "update",
		flag(params.ProjectIDFlag), "1234",
		flag(params.SimilarityIDFlag), "1234",
		flag(params.StateFlag), "URGENT",
		flag(params.SeverityFlag), "High",
		flag(params.CommentFlag), "",
		flag(params.ScanTypeFlag), "kics",
	)

	fmt.Println(err)
	assertError(t, err, "Predicate bad request")

	_, responseKics := executeCommand(
		t, "triage", "show",
		flag(params.ProjectIDFlag), "1234",
		flag(params.SimilarityIDFlag), "abcd",
		flag(params.ScanTypeFlag), "kics",
	)
	kicsPredicate := wrappers.PredicatesCollectionResponseModel{}
	_ = unmarshall(t, responseKics, &kicsPredicate, "Reading predicate should pass")
	assert.Assert(t, kicsPredicate.TotalCount == 0, "Predicate with invalid values should have 0 as the result.")
}
