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

func TestTriageShowAndUpdateWithCustomStates(t *testing.T) {
	t.Skip("Skipping this test temporarily until the API becomes available in the DEU environment.")
	fmt.Println("Step 1: Testing the command 'triage show' with predefined values.")
	// After the api/custom-states becomes available in the DEU environment, replace the hardcoded value with getRootScan(t) and create "state2" as a custom state in DEU.
	projectID := "a2d52ac9-d007-4d95-a65e-bbe61c3451a4"
	similarityID := "-1213859962"
	scanType := "sast"

	outputBuffer := executeCmdNilAssertion(
		t, "Fetching predicates should work.", "triage", "show",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.ScanTypeFlag), scanType,
	)

	predicateResult := []wrappers.Predicate{}
	fmt.Println(outputBuffer)
	_ = unmarshall(t, outputBuffer, &predicateResult, "Reading results should pass")
	assert.Assert(t, len(predicateResult) >= 1, "Should have at least 1 predicate as the result.")

	fmt.Println("Step 2: Updating the predicate state.")
	newState := "state2"
	newSeverity := "HIGH"
	comment := ""

	err, _ := executeCommand(
		t, "triage", "update",
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.StateFlag), newState,
		flag(params.SeverityFlag), newSeverity,
		flag(params.CommentFlag), comment,
		flag(params.ScanTypeFlag), scanType,
	)

	assert.NilError(t, err, "Updating the predicate should pass.")

	fmt.Println("Step 3: Fetching predicates again to validate update.")
	outputBufferAfterUpdate := executeCmdNilAssertion(
		t, "Fetching predicates after update should work.", "triage", "show",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.ScanTypeFlag), scanType,
	)

	updatedPredicateResult := []wrappers.Predicate{}
	fmt.Println(outputBufferAfterUpdate)
	_ = unmarshall(t, outputBufferAfterUpdate, &updatedPredicateResult, "Reading updated results should pass")
	assert.Assert(t, len(updatedPredicateResult) >= 1, "Should have at least 1 predicate after update.")

	found := false
	for _, pred := range updatedPredicateResult {
		if pred.SimilarityID == similarityID && pred.State == newState {
			found = true
			break
		}
	}

	assert.Assert(t, found, "Updated predicate should have state set to state2")
}
