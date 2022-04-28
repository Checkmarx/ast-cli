//go:build integration

package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

// Type for scan workflow response, used to assert the validity of the command's response
type ScanWorkflowResponse struct {
	Source      string    `json:"source"`
	Timestamp   time.Time `json:"timestamp"`
	Information string    `json:"info"`
}

// Create a scan with an empty project name
// Assert the scan fails with correct message
func TestScanCreateEmptyProjectName(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(params.ProjectName), "",
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), "sast",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Project name is required") // Creating a scan with empty project name should fail
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
		pollScanUntilStatus(t, scanID, wrappers.ScanCompleted, FullScanWait, ScanPollSleep),
		"Polling should complete",
	)

	executeScanAssertions(t, projectID, scanID, map[string]string{})
}

// Test ScaResolver as argument , this is a nop test
func TestScaResolverArg(t *testing.T) {
	scanID, projectID := createScanScaWithResolver(
		t,
		Dir,
		map[string]string{},
		"sast,kics",
		viper.GetString(resolverEnvVar),
	)
	defer deleteProject(t, projectID)
	assert.Assert(
		t,
		pollScanUntilStatus(t, scanID, wrappers.ScanCompleted, FullScanWait, ScanPollSleep),
		"Polling should complete when resolver used.",
	)
	executeScanAssertions(t, projectID, scanID, map[string]string{})
}

// Test ScaResolver as argument , no existing path to the resolver should fail
func TestScaResolverArgFailed(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(params.ProjectName), "resolver",
		flag(params.SourcesFlag), ".",
		flag(params.ScaResolverFlag), "./nonexisting",
		flag(params.ScanTypes), "sast,kics,sca",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "ScaResolver error")

	args = []string{
		"scan", "create",
		flag(params.ProjectName), "resolver",
		flag(params.SourcesFlag), ".",
		flag(params.ScaResolverFlag), viper.GetString(resolverEnvVar),
		flag(params.ScanTypes), "sast,kics,sca",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScaResolverParamsFlag), "-q --invalid-param \"invalid\"",
	}

	err, _ = executeCommand(t, args...)
	assertError(t, err, "ScaResolver error")
}

// Perform an initial scan with complete sources and an incremental scan with a smaller wait time
func TestIncrementalScan(t *testing.T) {
	projectName := getProjectNameForScanTests()

	scanID, projectID := createScanIncremental(t, Dir, projectName, map[string]string{})
	defer deleteProject(t, projectID)
	scanIDInc, projectIDInc := createScanIncremental(t, Dir, projectName, map[string]string{})

	assert.Assert(t, projectID == projectIDInc, "Project IDs should match")

	executeScanAssertions(t, projectID, scanID, map[string]string{})
	executeScanAssertions(t, projectIDInc, scanIDInc, map[string]string{})
}

// Start a scan guaranteed to take considerable time, cancel it and assert the status
func TestCancelScan(t *testing.T) {
	scanID, projectID := createScanSastNoWait(t, SlowRepo, map[string]string{})

	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	// canceling too quickly after creating fails the scan...
	time.Sleep(30 * time.Second)

	executeCmdNilAssertion(t, "Cancel should pass", "scan", "cancel", flag(params.ScanIDFlag), scanID)

	assert.Assert(t, pollScanUntilStatus(t, scanID, wrappers.ScanCanceled, 20, 5), "Scan should be canceled")
}

// Create a scan with the sources from the integration package, excluding go files and including zips
// Assert the scan completes
func TestScanCreateIncludeFilter(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), "sast",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.SourceDirFilterFlag), "!*go,!*Dockerfile,!*js",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "scan did not complete successfully") // Creating a scan with !*go,!*Dockerfile should fail
	args[11] = "*js"
	executeCmdWithTimeOutNilAssertion(t, "Including zip should fix the scan", 5*time.Minute, args...)
}

// Create a scan with the sources
// Assert the scan completes
func TestScanCreateWithThreshold(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sast",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.Threshold), "sast-high=1;sast-low=1;",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Threshold check finished with status Failed")
}

// Create a scan ignoring the exclusion of the .git directory
// Assert the folder is included in the logs
func TestScanCreateIgnoreExclusionFolders(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), "../..",
		flag(params.ScanTypes), "sast,sca",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.SourceDirFilterFlag), ".git,*.js", // needed one code file or the scan will end with partial code
		flag(params.BranchFlag), "dummy_branch",
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	_ = executeScanGetBuffer(t, args)
	reader := bufio.NewReader(&buf)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if strings.Contains(string(line), ".git") && !strings.Contains(
			string(line),
			".github",
		) && !strings.Contains(string(line), ".gitignore") {
			assert.Assert(
				t,
				!strings.Contains(string(line), "Excluded"),
				".Git directory and children should not be excluded",
			)
		}
		fmt.Printf("%s \n", line)
	}
}

// Test the timeout for a long scan
func TestScanTimeout(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), SlowRepo,
		flag(params.ScanTypes), "sast",
		flag(params.BranchFlag), "develop",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.ScanTimeoutFlag), "1",
	}

	cmd, buffer := createRedirectedTestCommand(t)
	err := execute(cmd, args...)

	assert.Assert(t, err != nil, "scan should time out")

	createdScan := wrappers.ScanResponseModel{}
	_ = unmarshall(t, buffer, &createdScan, "Reading scan response JSON should pass")

	assert.Assert(t, pollScanUntilStatus(t, createdScan.ID, wrappers.ScanCanceled, 120, 15), "Scan should be canceled")
}

func TestBrokenLinkScan(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), "sast",
		flag(params.BranchFlag), "main",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.IncludeFilterFlag), "broken_link.txt",
	}

	cmd := createASTIntegrationTestCommand(t)
	err := execute(cmd, args...)

	assertError(t, err, "broken_link.txt")
}

// Generic scan test execution
// - Get scan with 'scan list' and assert status and IDs
// - Get scan with 'scan show' and assert the ID
// - Assert all tags exist and are assigned to the scan
// - Delete the scan and assert it is deleted
func executeScanAssertions(t *testing.T, projectID, scanID string, tags map[string]string) {
	response := listScanByID(t, scanID)

	assert.Equal(t, len(response), 1, "Total scans should be 1")
	assert.Equal(t, response[0].ID, scanID, "Scan ID should match the created scan's ID")
	assert.Equal(t, response[0].ProjectID, projectID, "Project ID should match the created scan's project ID")
	assert.Assert(t, response[0].Status == wrappers.ScanCompleted, "Scan should be completed")

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
	return executeCreateScan(t, getCreateArgs(source, tags, "sast,kics"))
}

func createScanNoWait(t *testing.T, source string, tags map[string]string) (string, string) {
	return executeCreateScan(t, append(getCreateArgs(source, tags, "sast,kics"), flag(params.AsyncFlag)))
}

func createScanSastNoWait(t *testing.T, source string, tags map[string]string) (string, string) {
	return executeCreateScan(t, append(getCreateArgs(source, tags, "sast"), flag(params.AsyncFlag)))
}

// Create sca scan with resolver
func createScanScaWithResolver(
	t *testing.T,
	source string,
	tags map[string]string,
	scanTypes string,
	resolver string,
) (string, string) {
	return executeCreateScan(
		t,
		append(
			getCreateArgs(source, tags, scanTypes),
			flag(params.AsyncFlag),
			flag(params.ScaResolverFlag),
			resolver,
			flag(params.ScaResolverParamsFlag),
			"-q",
		),
	)
}

func createScanIncremental(t *testing.T, source string, name string, tags map[string]string) (string, string) {
	return executeCreateScan(t, append(getCreateArgsWithName(source, tags, name, "sast,kics"), "--sast-incremental"))
}

func getProjectNameForScanTests() string {
	return getProjectNameForTest() + "_for_scan"
}

func getCreateArgs(source string, tags map[string]string, scanTypes string) []string {
	projectName := getProjectNameForScanTests()
	return getCreateArgsWithName(source, tags, projectName, scanTypes)
}

func getCreateArgsWithName(source string, tags map[string]string, projectName, scanTypes string) []string {
	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), source,
		flag(params.ScanTypes), scanTypes,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.TagList), formatTags(tags),
		flag(params.BranchFlag), SlowRepoBranch,
	}
	return args
}

func executeCreateScan(t *testing.T, args []string) (string, string) {
	buffer := executeScanGetBuffer(t, args)

	createdScan := wrappers.ScanResponseModel{}
	_ = unmarshall(t, buffer, &createdScan, "Reading scan response JSON should pass")

	assert.Assert(t, createdScan.Status != wrappers.ScanFailed && createdScan.Status != wrappers.ScanCanceled)

	log.Println("Created new project with id: ", createdScan.ProjectID)
	log.Println("Created new scan with id: ", createdScan.ID)

	return createdScan.ID, createdScan.ProjectID
}

func executeScanGetBuffer(t *testing.T, args []string) *bytes.Buffer {
	return executeCmdWithTimeOutNilAssertion(t, "Creating a scan should pass", 5*time.Minute, args...)
}

func deleteScan(t *testing.T, scanID string) {
	log.Println("Deleting the scan with id ", scanID)
	executeCmdNilAssertion(t, "Deleting a scan should pass", "scan", "delete", flag(params.ScanIDFlag), scanID)
}

func listScanByID(t *testing.T, scanID string) []wrappers.ScanResponseModel {
	scanFilter := fmt.Sprintf("scan-ids=%s", scanID)

	outputBuffer := executeCmdNilAssertion(
		t,
		"Getting the scan should pass",
		"scan", "list", flag(params.FormatFlag), printer.FormatJSON, flag(params.FilterFlag), scanFilter,
	)

	// Read response from buffer
	var scanList []wrappers.ScanResponseModel
	_ = unmarshall(t, outputBuffer, &scanList, "Reading scan response JSON should pass")

	return scanList
}

func showScan(t *testing.T, scanID string) wrappers.ScanResponseModel {
	outputBuffer := executeCmdNilAssertion(
		t, "Getting the scan should pass", "scan", "show",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ScanIDFlag), scanID,
	)

	// Read response from buffer
	scan := wrappers.ScanResponseModel{}
	_ = unmarshall(t, outputBuffer, &scan, "Reading scan response JSON should pass")

	return scan
}

func pollScanUntilStatus(t *testing.T, scanID string, requiredStatus wrappers.ScanStatus, timeout, sleep int) bool {
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
			scan := showScan(t, scanID)
			log.Printf("Scan %s status %s\n", scanID, scan.Status)
			if s := string(scan.Status); s == string(requiredStatus) {
				return true
			} else if s == wrappers.ScanFailed || s == wrappers.ScanCanceled ||
				s == wrappers.ScanCompleted {
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

	buffer := executeCmdNilAssertion(
		t, "Workflow should pass", "scan", "workflow",
		flag(params.ScanIDFlag), scanID,
		flag(params.FormatFlag), printer.FormatJSON,
	)

	var workflow []ScanWorkflowResponse
	_ = unmarshall(t, buffer, &workflow, "Reading workflow output should work")

	assert.Assert(t, len(workflow) > 0, "At least one item should exist in the workflow response")
}

func TestScanLogsSAST(t *testing.T) {
	scanID, _ := getRootScan(t)

	executeCmdNilAssertion(
		t, "Getting scan SAST log should pass",
		"scan", "logs",
		flag(params.ScanIDFlag), scanID,
		flag(params.ScanTypeFlag), "sast",
	)
}

func TestScanLogsKICS(t *testing.T) {
	scanID, _ := getRootScan(t)

	executeCmdNilAssertion(
		t, "Getting scan KICS log should pass",
		"scan", "logs",
		flag(params.ScanIDFlag), scanID,
		flag(params.ScanTypeFlag), "kics",
	)
}

func TestPartialScanWithWrongPreset(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.PresetName), "Checkmarx Invalid",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "scan completed partially")
}

func TestFailedScanWithWrongPreset(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sast",
		flag(params.PresetName), "Checkmarx Invalid",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "scan did not complete successfully")
}

func TestScanCreateWithSSHKey(t *testing.T) {
	_ = viper.BindEnv("CX_SCAN_SSH_KEY")
	sshKey := viper.GetString("CX_SCAN_SSH_KEY")

	_ = ioutil.WriteFile(SSHKeyFilePath, []byte(sshKey), 0644)

	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), SSHRepo,
		flag(params.BranchFlag), "main",
		flag(params.SSHKeyFlag), SSHKeyFilePath,
	}

	executeCmdWithTimeOutNilAssertion(t, "Create a scan with ssh-key should pass", 4*time.Minute, args...)

	_ = os.Remove(SSHKeyFilePath)
}

func retrieveResultsFromScanId(t *testing.T, scanId string) (wrappers.ScanResultsCollection, error) {
	resultsArgs := []string{
		"results",
		"show",
		flag(params.ScanIDFlag),
		scanId,
	}
	executeCmdNilAssertion(t, "Getting results should pass", resultsArgs...)
	file, err := ioutil.ReadFile("cx_result.json")
	defer func() {
		os.Remove("cx_result.json")
	}()
	if err != nil {
		return wrappers.ScanResultsCollection{}, err
	}
	results := wrappers.ScanResultsCollection{}
	_ = json.Unmarshal([]byte(file), &results)
	return results, err
}

func TestScanWorkFlowWithSastEngineFilter(t *testing.T) {
	insecurePath := "data/insecure.zip"
	args := getCreateArgsWithName(insecurePath, Tags, getProjectNameForScanTests(), "sast")
	args = append(args, flag(params.SastFilterFlag), "!*.java")
	scanId, projectId := executeCreateScan(t, args)
	assert.Assert(t, scanId != "", "Scan ID should not be empty")
	assert.Assert(t, projectId != "", "Project ID should not be empty")
	results, err := retrieveResultsFromScanId(t, scanId)
	assert.Assert(t, err == nil, "Results retrieved should not throw an error")
	for _, result := range results.Results {
		for _, node := range result.ScanResultData.Nodes {
			assert.Assert(t, !strings.HasSuffix(node.FileName, "java"), "File name should not contain java")
		}
	}
}

func TestScanCreateWithSSHKey(t *testing.T) {
	_ = viper.BindEnv("CX_SCAN_SSH_KEY")
	sshKey := viper.GetString("CX_SCAN_SSH_KEY")

	_ = ioutil.WriteFile(SSHKeyFilePath, []byte(sshKey), 0644)

	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), SSHRepo,
		flag(params.BranchFlag), "main",
		flag(params.SSHKeyFlag), SSHKeyFilePath,
	}

	executeCmdWithTimeOutNilAssertion(t, "Create a scan with ssh-key should pass", 4*time.Minute, args...)

	_ = os.Remove(SSHKeyFilePath)
}
