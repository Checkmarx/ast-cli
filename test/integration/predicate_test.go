//go:build integration

package integration

import (
	"io"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

// - KICS : Get all predicates for a given project and similarity id
// - SAST : Get all predicates for a given project and similarity id
// - KICS : Update Severity for a given similarity id
// - SAST : Update Severity for a given similarity id
// - KICS : Update Status for a given similarity id
// - SAST : Update Status for a given similarity id

//func TestKicsGetPredicatesForSimilarityId(t *testing.T) {
//	//assertRequiredParameter(t, "Project name is required", "triage", "show")
//
//	_, projectId := getRootScan(t)
//	similarityId := "4fb227cd0a8f6af8e8646679411e8ec7cd14f17f8a2f1c33515fb3a450f3e050"
//	scanType := "kics"
//	// triage show --project-id "c184dbea-ba31-4b6c-bbb3-65be058281e7" --similarity-id "4fb227c3e050" --scan-type "kics"
//	outputBuffer := executeCmdNilAssertion(t, "triage", "show",
//		flag(params.FormatFlag), util.FormatJSON,
//		flag(params.ProjectIDFlag), projectId,
//		flag(params.SimilarityIdFlag), similarityId,
//		flag(params.ScanTypeFlag), scanType)
//
//	result := wrappers.PredicatesCollectionResponseModel{}
//	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")
//
//	assert.Assert(t, uint(len(result.PredicateHistoryPerProject)) == result.TotalCount, "Should have results")
//
//}

func TestSastUpdatePredicatesForSimilarityId(t *testing.T) {
	//assertRequiredParameter(t, "Project name is required", "triage", "show")

	_, projectID := getRootScan(t)
	similarityID := "1826563305"
	state := "Urgent"
	severity := "Medium"
	comment := "Testing CLI Command for triage."
	scanType := "sast"

	// triage show --project-id "c184dbea-ba31-4b6c-bbb3-65be058281e7" --similarity-id "4fb227c3e050" --scan-type "kics"
	outputBuffer := executeCmdNilAssertion(t, "triage", "update",
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.StateFlag), state,
		flag(params.SeverityFlag), severity,
		flag(params.CommentFlag), comment,
		flag(params.ScanTypeFlag), scanType)

	_, readingError := io.ReadAll(outputBuffer)
	assert.NilError(t, readingError, "Reading result should pass")

}

func TestSastGetPredicatesForSimilarityId(t *testing.T) {
	//assertRequiredParameter(t, "Project name is required", "triage", "show")

	_, projectID := getRootScan(t)
	similarityID := "1826563305"
	scanType := "sast"
	// triage show --project-id "c184dbea-ba31-4b6c-bbb3-65be058281e7" --similarity-id "4fb227c3e050" --scan-type "kics"
	outputBuffer := executeCmdNilAssertion(t, "triage", "show",
		flag(params.FormatFlag), util.FormatJSON,
		flag(params.ProjectIDFlag), projectID,
		flag(params.SimilarityIDFlag), similarityID,
		flag(params.ScanTypeFlag), scanType)

	result := wrappers.PredicatesCollectionResponseModel{}
	_ = unmarshall(t, outputBuffer, &result, "Reading results should pass")

	assert.Assert(t, (len(result.PredicateHistoryPerProject)) == result.TotalCount, "Should have results")

}
