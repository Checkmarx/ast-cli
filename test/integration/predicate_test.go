//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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

	index := 0
	for i := range result.Results {
		if strings.EqualFold(result.Results[i].Type, params.SastType) {
			index = i
			break
		}
	}

	similarityID := result.Results[index].SimilarityID

	state := "CONFIRMED"
	if !strings.EqualFold(result.Results[index].State, "Urgent") {
		state = "URGENT"
	}
	severity := "HIGH"
	if !strings.EqualFold(result.Results[index].Severity, "Medium") {
		severity = "MEDIUM"
	}
	comment := "Testing CLI Command for triage."
	scanType := result.Results[index].Type

	args := []string{
		"triage", "update",
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.StateFlag), state,
		flag(params.SeverityFlag), severity,
		flag(params.CommentFlag), comment,
		flag(params.ScanTypeFlag), scanType,
	}

	err, outputBufferForStep1 := executeCommand(t, args...)
	_, readingError := io.ReadAll(outputBufferForStep1)
	assert.NilError(t, readingError, "Reading result should pass")

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

	assert.Assert(t, (len(predicateResult)) >= 1, "Should have at least 1 predicate as the result.")

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



func TestSastUpdateAndGetPredicateWithCustomStateID(t *testing.T) {
	t.Skip("Skipping TestSastUpdateAndGetPredicateWithCustomStateID for now")

	fmt.Println("Step 1: Testing the command 'triage update' with custom state-id to update an issue from the project.")

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

	index := 0
	for i := range result.Results {
		if strings.EqualFold(result.Results[i].Type, params.SastType) {
			index = i
			break
		}
	}

	similarityID := result.Results[index].SimilarityID
	scanType := result.Results[index].Type

	severity := "HIGH"
	comment := "Testing CLI Command with custom state-id."

	args := []string{
		"triage", "update",
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.CustomStateIDFlag), "324",
		flag(params.StateFlag), "triageTest",
		flag(params.SeverityFlag), severity,
		flag(params.CommentFlag), comment,
		flag(params.ScanTypeFlag), scanType,
	}

	err, outputBufferForStep1 := executeCommand(t, args...)
	_, readingError := io.ReadAll(outputBufferForStep1)
	assert.NilError(t, readingError, "Reading result should pass")
	assert.NilError(t, err, "Updating the predicate with custom state-id should pass.")

	fmt.Println("Step 2: Testing the command 'triage show' to verify the updated predicate with state-id.")
	outputBufferForStep2 := executeCmdNilAssertion(
		t, "Predicates should be fetched.", "triage", "show",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.ScanTypeFlag), scanType,
	)

	predicateResult := []wrappers.Predicate{}
	_ = unmarshall(t, outputBufferForStep2, &predicateResult, "Reading results should pass")


	assert.Assert(t, len(predicateResult) >= 1, "Should have at least 1 predicate as the result.")
	found := false
	for _, predicate := range predicateResult {
		if predicate.StateID == 324 {
			assert.Equal(t, predicate.State, "triageTest", "The state name for state-id 324 should be updated to triageTest")
			found = true
			break
		}
	}
	assert.Assert(t, found, "Predicate with state-id 324 should be found in the results")
}