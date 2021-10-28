//go:build integration

package integration

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/checkmarxDev/ast-cli/internal/commands/util"
	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/google/uuid"

	scansApi "github.com/checkmarxDev/scans/pkg/api/scans"
	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/rest/v1"
	"gotest.tools/assert"
)

// Type for scan workflow response, used to assert the validity of the command's response
type ScanWorkflowResponse struct {
	Source      string    `json:"source"`
	Timestamp   time.Time `json:"timestamp"`
	Information string    `json:"info"`
}

// Create scans from current dir, zip and url and perform assertions in executeScanAssertions
func TestScansE2E(t *testing.T) {

	scanID, projectID := createScan(t, Zip, Tags)
	defer deleteProject(t, projectID)

	executeScanAssertions(t, projectID, scanID, Tags)
}

// Perform a nowait scan and poll status until completed
func TestNoWaitScan(t *testing.T) {
	scanID, projectID := createScanNoWait(t, Dir, map[string]string{})
	defer deleteProject(t, projectID)

	assert.Assert(
		t,
		pollScanUntilStatus(t, scanID, scansApi.ScanCompleted, FullScanWait, ScanPollSleep),
		"Polling should complete",
	)

	executeScanAssertions(t, projectID, scanID, map[string]string{})
}

// Test ScaResolver environment variable, this is a nop test
func TestScaResolverEnv(t *testing.T) {
	scanID, projectID := createScanNoWaitWithResolver(t, Dir, map[string]string{})
	defer deleteProject(t, projectID)
	assert.Assert(
		t,
		pollScanUntilStatus(t, scanID, scansApi.ScanCompleted, FullScanWait, ScanPollSleep),
		"Polling should complete when resolver used.",
	)
	executeScanAssertions(t, projectID, scanID, map[string]string{})
}

// Perform an initial scan with complete sources and an incremental scan with a smaller wait time
func TestIncrementalScan(t *testing.T) {
	projectName := fmt.Sprintf("integration_test_incremental_%s", uuid.New().String())

	scanID, projectID := createScanIncremental(t, Dir, projectName, map[string]string{})
	defer deleteProject(t, projectID)
	scanIDInc, projectIDInc := createScanIncremental(t, Dir, projectName, map[string]string{})

	assert.Assert(t, projectID == projectIDInc, "Project IDs should match")

	executeScanAssertions(t, projectID, scanID, map[string]string{})
	executeScanAssertions(t, projectIDInc, scanIDInc, map[string]string{})
}

// Start a scan guaranteed to take considerable time, cancel it and assert the status
func TestCancelScan(t *testing.T) {
	scanID, projectID := createScanNoWait(t, SlowRepo, map[string]string{})

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	executeCmdNilAssertion(t, "Cancel should pass", "scan", "cancel", flag(params.ScanIDFlag), scanID)

	assert.Assert(t, pollScanUntilStatus(t, scanID, scansApi.ScanCanceled, 20, 5), "Scan should be canceled")
}

// Create a scan with the sources from the integration package, excluding go files and including zips
// Assert the scan completes
func TestScanCreateIncludeFilter(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), "sast, sca",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.SourceDirFilterFlag), "!*go,!*Dockerfile",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "scan did not complete successfully") // Creating a scan with !*go,!*Dockerfile should fail

	args = append(args, flag(params.IncludeFilterFlag), "*txt")
	executeCmdWithTimeOutNilAssertion(t, "Including zip should fix the scan", 5*time.Minute, args...)
}

// Generic scan test execution
// - Get scan with 'scan list' and assert status and IDs
// - Get scan with 'scan show' and assert the ID
// - Assert all tags exist and are assigned to the scan
// - Delete the scan and assert it is deleted
func executeScanAssertions(t *testing.T, projectID string, scanID string, tags map[string]string) {
	response := listScanByID(t, scanID)

	assert.Equal(t, len(response), 1, "Total scans should be 1")
	assert.Equal(t, response[0].ID, scanID, "Scan ID should match the created scan's ID")
	assert.Equal(t, response[0].ProjectID, projectID, "Project ID should match the created scan's project ID")
	assert.Assert(t, response[0].Status == scansApi.ScanCompleted, "Scan should be completed")

	scan := showScan(t, scanID)
	assert.Equal(t, scan.ID, scanID, "Scan ID should match the created scan's ID")

	allTags := getAllTags(t, "scan")
	for key := range tags {
		_, ok := allTags[key]
		assert.Assert(t, ok, "Get all tags response should contain all created tags. Missing %s", key)

		val, ok := scan.Tags[key]
		assert.Assert(t, ok, "Scan should contain all created tags. Missing %s", key)
		assert.Equal(t, val, Tags[key], "Tag value should be equal")
	}

	deleteScan(t, scanID)

	response = listScanByID(t, scanID)

	assert.Equal(t, len(response), 0, "Total scans should be 0 as the scan was deleted")
}

func createScan(t *testing.T, source string, tags map[string]string) (string, string) {
	return executeCreateScan(t, getCreateArgs(source, tags))
}

func createScanNoWait(t *testing.T, source string, tags map[string]string) (string, string) {
	return executeCreateScan(t, append(getCreateArgs(source, tags), "--nowait"))
}

func createScanNoWaitWithResolver(t *testing.T, source string, tags map[string]string) (string, string) {
	return executeCreateScan(t, append(getCreateArgs(source, tags), "--nowait", "--sca-resolver", "nop"))
}

func createScanIncremental(t *testing.T, source string, name string, tags map[string]string) (string, string) {
	return executeCreateScan(t, append(getCreateArgsWithName(source, tags, name), "--sast-incremental"))
}

func getCreateArgs(source string, tags map[string]string) []string {
	projectName := fmt.Sprintf("integration_test_scan_%s", uuid.New().String())
	return getCreateArgsWithName(source, tags, projectName)
}

func getCreateArgsWithName(source string, tags map[string]string, projectName string) []string {
	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), source,
		flag(params.ScanTypes), "sast,kics",
		flag(params.FormatFlag), util.FormatJSON,
		flag(params.TagList), formatTags(tags),
		flag(params.BranchFlag), "dummy_branch",
	}
	return args
}

func executeCreateScan(t *testing.T, args []string) (string, string) {

	buffer := executeCmdWithTimeOutNilAssertion(t, "Creating a scan should pass", 5*time.Minute, args...)

	createdScan := scansRESTApi.ScanResponseModel{}
	_ = unmarshall(t, buffer, &createdScan, "Reading scan response JSON should pass")

	assert.Assert(t, createdScan.Status != scansApi.ScanFailed && createdScan.Status != scansApi.ScanCanceled)

	log.Printf("Scan ID %s created in test", createdScan.ID)

	return createdScan.ID, createdScan.ProjectID
}

func deleteScan(t *testing.T, scanID string) {
	executeCmdNilAssertion(t, "Deleting a scan should pass", "scan", "delete", flag(params.ScanIDFlag), scanID)
}

func listScanByID(t *testing.T, scanID string) []scansRESTApi.ScanResponseModel {
	scanFilter := fmt.Sprintf("scan-ids=%s", scanID)

	outputBuffer := executeCmdNilAssertion(t,
		"Getting the scan should pass",
		"scan", "list", flag(params.FormatFlag), util.FormatJSON, flag(params.FilterFlag), scanFilter,
	)

	// Read response from buffer
	var scanList []scansRESTApi.ScanResponseModel
	_ = unmarshall(t, outputBuffer, &scanList, "Reading scan response JSON should pass")

	return scanList
}

func showScan(t *testing.T, scanID string) scansRESTApi.ScanResponseModel {
	outputBuffer := executeCmdNilAssertion(t, "Getting the scan should pass", "scan", "show",
		flag(params.FormatFlag), util.FormatJSON,
		flag(params.ScanIDFlag), scanID,
	)

	// Read response from buffer
	scan := scansRESTApi.ScanResponseModel{}
	_ = unmarshall(t, outputBuffer, &scan, "Reading scan response JSON should pass")

	return scan
}

func pollScanUntilStatus(t *testing.T, scanID string, requiredStatus scansApi.ScanStatus, timeout, sleep int) bool {
	log.Printf("Set timeout of %d seconds for the scan to complete...\n", timeout)
	// Wait for the scan to finish. See it's completed successfully
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return false
		default:
			log.Printf("Polling scan %s\n", scanID)
			scan := listScanByID(t, scanID)
			if s := string(scan[0].Status); s == string(requiredStatus) {
				return true
			} else if s == scansApi.ScanFailed || s == scansApi.ScanCanceled ||
				s == scansApi.ScanCompleted {
				return false
			} else {
				time.Sleep(time.Duration(sleep) * time.Second)
			}
		}
	}
}

// Get a scan workflow and assert its structure
func TestScanWorkflow(t *testing.T) {
	scanID, _ := getRootScan(t)

	buffer := executeCmdNilAssertion(t, "Workflow should pass", "scan", "workflow",
		flag(params.ScanIDFlag), scanID,
		flag(params.FormatFlag), util.FormatJSON,
	)

	var workflow []ScanWorkflowResponse
	_ = unmarshall(t, buffer, &workflow, "Reading workflow output should work")

	assert.Assert(t, len(workflow) > 0, "At least one item should exist in the workflow response")
}

func TestScanLogsSAST(t *testing.T) {
	scanID, _ := getRootScan(t)

	executeCmdNilAssertion(t, "Getting scan SAST log should pass",
		"scan", "logs",
		flag(params.ScanIDFlag), scanID,
		flag(params.ScanTypeFlag), "sast",
	)
}

func TestScanLogsKICS(t *testing.T) {
	scanID, _ := getRootScan(t)

	executeCmdNilAssertion(t, "Getting scan KICS log should pass",
		"scan", "logs",
		flag(params.ScanIDFlag), scanID,
		flag(params.ScanTypeFlag), "kics",
	)
}
