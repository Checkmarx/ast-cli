//go:build integration

package integration

import (
	"fmt"
	"io"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

var projectID string

func TestSastUpdateAndGetPredicatesForSimilarityId(t *testing.T) {

	fmt.Println("Step 1: Testing the command 'triage update' to update an issue from the project.")

	_, projectID = getRootScan(t)
	similarityID := "1826563305"
	state := "Urgent"
	severity := "Medium"
	comment := "Testing CLI Command for triage."
	scanType := "sast"

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

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Updating the predicate with same values should pass.")

	fmt.Println("Step 2: Testing the command 'triage show' to get the same predicate back.")
	outputBufferForStep2 := executeCmdNilAssertion(
		t, "Predicates should be fetched.", "triage", "show",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.ScanTypeFlag), scanType,
	)

	result := []wrappers.Predicate{}
	fmt.Println(outputBufferForStep2)
	_ = unmarshall(t, outputBufferForStep2, &result, "Reading results should pass")

	assert.Assert(t, (len(result)) == 1, "Should have 1 predicate as the result.")

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
