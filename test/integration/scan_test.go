//go:build integration

package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands"
	realtime "github.com/checkmarx/ast-cli/internal/commands/scarealtime"
	"github.com/checkmarx/ast-cli/internal/commands/scarealtime/scaconfig"
	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	exitCodes "github.com/checkmarx/ast-cli/internal/constants/exit-codes"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	asserts "github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
	"gotest.tools/assert"
)

const (
	fileSourceValue       = "data/Dockerfile"
	fileSourceValueVul    = "data/mock.dockerfile"
	engineValue           = "docker"
	additionalParamsValue = "-v"
	scanCommand           = "scan"
	kicsRealtimeCommand   = "kics-realtime"
	invalidEngineValue    = "invalidEngine"
	scanList              = "list"
	projectIDParams       = "project-id="
	scsRepoURL            = "https://github.com/CheckmarxDev/easybuggy-scs-tests"
	invalidClientID       = "invalidClientID"
	invalidClientSecret   = "invalidClientSecret"
	invalidAPIKey         = "invalidAPI"
	invalidTenant         = "invalidTenant"
	timeout               = 10 * time.Minute
)

var (
	_, b, _, _       = runtime.Caller(0)
	projectDirectory = filepath.Dir(b)
	scsRepoToken     = getScsRepoToken()
)

// Type for scan workflow response, used to assert the validity of the command's response
type ScanWorkflowResponse struct {
	Source      string    `json:"source"`
	Timestamp   time.Time `json:"timestamp"`
	Information string    `json:"info"`
}

func TestCreateScan_WithOnlyValidApikeyFlag_Success(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       invalidAPIKey,
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
		flag(params.AstAPIKeyFlag), originals[params.AstAPIKeyEnv],
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestCreateScan_WithOnlyValidApikeyEnvVar_Success(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestCreateScan_WithOnlyInvalidApikeyEnvVar_Fail(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       invalidAPIKey,
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, "Error validating scan types: Token decoding error: token is malformed: token contains an invalid number of segments")
}

func TestCreateScan_WithOnlyInvalidApikeyFlag_Fail(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       "",
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
		flag(params.AstAPIKeyFlag), "invalid_apikey",
	}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, "Error validating scan types: Token decoding error: token is malformed: token contains an invalid number of segments")
}

func TestCreateScan_WithValidClientCredentialsFlag_Success(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       "",
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
		flag(params.AccessKeyIDFlag), originals[params.AccessKeyIDEnv],
		flag(params.AccessKeySecretFlag), originals[params.AccessKeySecretEnv],
		flag(params.TenantFlag), originals[params.TenantEnv],
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestCreateScan_WithInvalidClientCredentialsFlag_Fail(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       invalidAPIKey,
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
		flag(params.AccessKeyIDFlag), "invalid_client_ID",
		flag(params.AccessKeySecretFlag), "invalid_client_secret",
	}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, "Error validating scan types: Token decoding error: token is malformed: token contains an invalid number of segments")
}

func TestCreateScan_WithValidClientCredentialsEnvVars_Success(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv: "",
	})

	defer setEnvVars(originals)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestCreateScan_WithInvalidClientCredentialsEnvVars_Fail(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       "",
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, "Error validating scan types: 404 Provided Tenant Name is invalid \n")
}

func getOriginalEnvVars() map[string]string {
	return map[string]string{
		params.AstAPIKeyEnv:       os.Getenv(params.AstAPIKeyEnv),
		params.AccessKeyIDEnv:     os.Getenv(params.AccessKeyIDEnv),
		params.AccessKeySecretEnv: os.Getenv(params.AccessKeySecretEnv),
		params.TenantEnv:          os.Getenv(params.TenantEnv),
	}
}

func setEnvVars(envVars map[string]string) {
	for key, value := range envVars {
		os.Setenv(key, value)
	}
}

// Create a scan with an empty project name
// Assert the scan fails with correct message
func TestScanCreateEmptyProjectName(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(params.ProjectName), "",
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), params.SastType,
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Project name is required") // Creating a scan with empty project name should fail
}

func TestScanCreate_ExistingApplicationAndExistingProject_CreateScanSuccessfully(t *testing.T) {
	_, projectName := createNewProject(t, nil, nil, GenerateRandomProjectNameForScan())
	args := []string{
		"scan", "create",
		flag(params.ApplicationName), "my-application",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
		flag(params.DebugFlag),
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestScanCreate_FolderWithSymbolicLinkWithAbsolutePath_CreateScanSuccessfully(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/project-with-directory-symlink",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestScanCreate_FolderWithSymbolicLinkWithRelativePath_CreateScanSuccessfully(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/symlink-relative-path-folder",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestScanCreate_ExistingApplicationAndNotExistingProject_CreatingNewProjectAndCreateScanSuccessfully(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(params.ApplicationName), "my-application",
		flag(params.ProjectName), GenerateRandomProjectNameForScan(),
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, projectID := executeCreateScan(t, args)
	assert.Assert(t, scanID != "", "Scan ID should not be empty")
	assert.Assert(t, projectID != "", "Project ID should not be empty")
}

func TestScanCreate_WithNewProjectAndApplicationDoesntExist_ShouldFailScanWithError(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(params.ApplicationName), "application-that-doesnt-exist",
		flag(params.ProjectName), "newProject",
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, errorConstants.ApplicationDoesntExistOrNoPermission)
}

func TestContainerEngineScansE2E_ContainerImagesFlagAndScanType(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/Dockerfile-mysql571.zip",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ContainerImagesFlag), "nginx:alpine,debian:9",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	if isFFEnabled(t, wrappers.ContainerEngineCLIEnabled) {
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty")
		assert.Assert(t, projectID != "", "Project ID should not be empty")
	}
}

func TestContainerEngineScansE2E_ContainerImagesFlagOnly(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ContainerImagesFlag), "nginx:alpine",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	if isFFEnabled(t, wrappers.ContainerEngineCLIEnabled) {
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty")
		assert.Assert(t, projectID != "", "Project ID should not be empty")
	}
}

func TestContainerEngineScansE2E_ContainerImagesAndDebugFlags(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ContainerImagesFlag), "mysql:5.7",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.DebugFlag),
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	if isFFEnabled(t, wrappers.ContainerEngineCLIEnabled) {
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty")
		assert.Assert(t, projectID != "", "Project ID should not be empty")
	}
}

func TestContainerEngineScansE2E_ContainerImagesFlagAndEmptyFolderProject(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.ContainerImagesFlag), "mysql:5.7",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.ScanTypes), params.ContainersTypeFlag,
	}
	if isFFEnabled(t, wrappers.ContainerEngineCLIEnabled) {
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty")
		assert.Assert(t, projectID != "", "Project ID should not be empty")
	}
}

func TestContainerEngineScansE2E_InvalidContainerImagesFlag(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/Dockerfile-mysql571.zip",
		flag(params.ContainerImagesFlag), "nginx:",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	if isFFEnabled(t, wrappers.ContainerEngineCLIEnabled) {
		err, _ := executeCommand(t, testArgs...)
		assertError(t, err, "Invalid value for --container-images flag. The value must be in the format <image-name>:<image-tag>")
	}
}

// Create scans from current dir, zip and url and perform assertions in executeScanAssertions
func TestScansE2E(t *testing.T) {
	cleanupCxZipFiles(t)
	scanID, projectID := executeCreateScan(t, getCreateArgsWithGroups(Zip, Tags, Groups, params.IacType))

	executeScanAssertions(t, projectID, scanID, Tags)
	glob, err := filepath.Glob(filepath.Join(os.TempDir(), "cx*.zip"))
	if err != nil {

		return
	}
	assert.Equal(t, len(glob), 0, "Zip file not removed")
}

func cleanupCxZipFiles(t *testing.T) {
	glob, err := filepath.Glob(filepath.Join(os.TempDir(), "cx*.zip"))
	if err != nil {
		t.Logf("Failed to search for cx*.zip files: %v", err)
	}
	for _, file := range glob {
		err = os.Remove(file)
		if err != nil {
			t.Logf("Failed to remove file %s: %v", file, err)
		}
	}
}

func TestFastScan(t *testing.T) {
	projectName := getProjectNameForScanTests()
	// Create a scan
	scanID, projectID := createScanWithFastScan(t, Dir, projectName, map[string]string{})
	executeScanAssertions(t, projectID, scanID, map[string]string{})
}

func createScanWithFastScan(t *testing.T, source string, name string, tags map[string]string) (string, string) {
	args := append(getCreateArgsWithName(source, tags, name, "sast"), flag(params.SastFastScanFlag))
	return executeCreateScan(t, args)
}

func TestScansUpdateProjectGroups(t *testing.T) {
	cleanupCxZipFiles(t)
	scanID, projectID := executeCreateScan(t, getCreateArgs(Zip, Tags, params.IacType), "timeout")
	response := listScanByID(t, scanID)
	scanID, projectID = executeCreateScan(t, getCreateArgsWithNameAndGroups(Zip, Tags, Groups, response[0].ProjectName, params.IacType), "timeout")

	executeScanAssertions(t, projectID, scanID, Tags)
	glob, err := filepath.Glob(filepath.Join(os.TempDir(), "cx*.zip"))
	if err != nil {

		return
	}
	assert.Equal(t, len(glob), 0, "Zip file not removed")
}

func TestInvalidSource(t *testing.T) {
	args := []string{scanCommand, "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "invalidSource",
		flag(params.ScanTypes), "sast",
		flag(params.BranchFlag), "dummy_branch"}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Failed creating a scan: Input in bad format: Sources input has bad format: invalidSource")
}

func TestScanShowRequiredOrInvalidScanId(t *testing.T) {
	args := []string{scanCommand, "show", flag(params.ScanIDQueryParam), ""}
	err, _ := executeCommand(t, args...)
	assert.Assert(t, strings.Contains(err.Error(), "Failed showing a scan: Please provide a scan ID"))
	args = []string{scanCommand, "show", flag(params.ScanIDQueryParam), "invalidScan"}
	err, _ = executeCommand(t, args...)
	assert.Assert(t, strings.Contains(err.Error(), "Failed showing a scan:"))
}

func TestRequiredScanIdToGetScanShow(t *testing.T) {
	args := []string{scanCommand, "workflow", flag(params.ScanIDQueryParam), ""}
	err, _ := executeCommand(t, args...)
	assert.Assert(t, strings.Contains(err.Error(), "Please provide a scan ID"))
}

// Test ScaResolver as argument , this is a nop test
func TestScaResolverArg(t *testing.T) {
	scanID, projectID := createScanScaWithResolver(
		t,
		Dir,
		map[string]string{},
		params.IacType,
		viper.GetString(resolverEnvVar),
	)

	assert.Assert(
		t,
		pollScanUntilStatus(t, scanID, wrappers.ScanCompleted, FullScanWait, ScanPollSleep),
		"Polling should complete when resolver used.",
	)
	executeScanAssertions(t, projectID, scanID, map[string]string{})
}

// Test ScaResolver as argument, no existing path to the resolver should fail
func TestScaResolverArgFailed(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), ".",
		flag(params.ScaResolverFlag), "./nonexisting",
		flag(params.ScanTypes), "iac-security,sca",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "ScaResolver error")

	args = []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), ".",
		flag(params.ScaResolverFlag), viper.GetString(resolverEnvVar),
		flag(params.ScanTypes), "iac-security,sca",
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
	scanIDInc, projectIDInc := createScanIncremental(t, Dir, projectName, map[string]string{})

	assert.Assert(t, projectID == projectIDInc, "Project IDs should match")

	executeScanAssertions(t, projectID, scanID, map[string]string{})
	executeScanAssertions(t, projectIDInc, scanIDInc, map[string]string{})
}

// Start a scan guaranteed to take considerable time, cancel it and assert the status
func TestCancelScan(t *testing.T) {
	scanID, _ := createScanSastNoWait(t, SlowRepo, map[string]string{})

	defer deleteScan(t, scanID)

	// canceling too quickly after creating fails the scan...
	time.Sleep(30 * time.Second)

	assert.Assert(t, pollScanUntilStatus(t, scanID, wrappers.ScanRunning, 200, 5), "Scan should be running before cancel")

	executeCmdNilAssertion(t, "Cancel should pass", "scan", "cancel", flag(params.ScanIDFlag), scanID)

	assert.Assert(t, pollScanUntilStatus(t, scanID, wrappers.ScanCanceled, 90, 5), "Scan should be canceled")
}

// Create a scan with the sources from the integration package, excluding go files and including zips
// Assert the scan completes
func TestScanCreateIncludeFilter(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), params.IacType,
		flag(params.PresetName), "Checkmarx Default",
		flag(params.SourceDirFilterFlag), "!*go,!*Dockerfile,!*js,!*json,!*tf",
		flag(params.IacsFilterFlag), "!Dockerfile",
		flag(params.BranchFlag), "dummy_branch",
	}

	args[11] = "*js"
	executeCmdWithTimeOutNilAssertion(t, "Including zip should fix the scan", timeout, args...)
}

// Create a scan with the sources
// Assert the scan completes
func TestScanCreateWithThresholdShouldBlock(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), params.SastType,
		flag(params.PresetName), "Checkmarx Default",
		flag(params.Threshold), "sast-critical=1;sast-high=1;sast-low=1;",
		flag(params.KicsFilterFlag), "!Dockerfile",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Threshold check finished with status Failed")
}

func TestScanCreateWithThreshold(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), params.IacType,
		flag(params.PresetName), "Checkmarx Default",
		flag(params.Threshold), "iac-security-high=100;",
		flag(params.KicsFilterFlag), "!Dockerfile",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "")
}

func TestScansAPISecThresholdShouldBlock(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/sources.zip",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.ScanTypes), "sast, api-security",
		flag(params.SastFastScanFlag),
		flag(params.Threshold), "api-security-high=1",
	}
	_, _ = executeCommand(t, testArgs...)
}

// Create a scan with the sources
// Assert the scan completes
func TestScanCreateWithThresholdParseError(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sca",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.Threshold), "sca-high=error;",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "")
}

// Create a scan with the sources
// Assert the scan completes
func TestScanCreateWithThresholdAndReportGenerate(t *testing.T) {
	_, projectName := getRootProject(t)

	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       "",
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sca",
		flag(params.SastRedundancyFlag),
		flag(params.PresetName), "Checkmarx Default",
		flag(params.Threshold), "sca-high=1",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.TargetFormatFlag), "json",
		flag(params.TargetPathFlag), "/tmp/",
		flag(params.TargetFlag), "results",
		flag(params.AstAPIKeyFlag), originals[params.AstAPIKeyEnv],
	}

	cmd := createASTIntegrationTestCommand(t)
	err := executeWithTimeout(cmd, 5*time.Minute, args...)
	assertError(t, err, "Threshold check finished with status Failed")

	file, fileError := os.Stat(fmt.Sprintf("%s%s.%s", "/tmp/", "results", "json"))
	assert.Assert(t, file.Size() > 5000, "Report file should be bigger than 5KB")
	assert.NilError(t, fileError, "Report file should exist for extension")
}

// Create a scan ignoring the exclusion of the .git directory
// Assert the folder is included in the logs
func TestScanCreateIgnoreExclusionFolders(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), "../..",
		flag(params.ScanTypes), params.IacType,
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
		flag(params.ScanTypes), "sca",
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
		flag(params.DebugFlag),
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	cmd := createASTIntegrationTestCommand(t)
	err := execute(cmd, args...)

	assert.NilError(t, err)

	output, err := io.ReadAll(&buf)

	assert.NilError(t, err)

	assert.Assert(t, strings.Contains(string(output), commands.DanglingSymlinkError))
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
	if isFFEnabled(t, wrappers.ContainerEngineCLIEnabled) {
		return executeCreateScan(t, getCreateArgs(source, tags, "sast , sca , iac-security , api-security,   container-security, scs"))
	} else {
		return executeCreateScan(t, getCreateArgs(source, tags, "sast , sca , iac-security , api-security, scs"))
	}
}

func createScanNoWait(t *testing.T, source string, tags map[string]string, projectName string) (string, string) {
	return executeCreateScan(t, append(getCreateArgsWithNameAndGroups(source, tags, nil, projectName, " sast , sca,iac-security "), flag(params.AsyncFlag)))
}

func createScanSastNoWait(t *testing.T, source string, tags map[string]string) (string, string) {
	return executeCreateScan(t, append(getCreateArgs(source, tags, "sast,sca"), flag(params.AsyncFlag)))
}
func createScanWithEngines(t *testing.T, source string, tags map[string]string, scanTypes string) (string, string) {
	return executeCreateScan(t, append(getCreateArgs(source, tags, scanTypes), flag(params.AsyncFlag)))
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
	return executeCreateScan(t, append(getCreateArgsWithName(source, tags, name, "sast,sca,iac-security"), "--sast-incremental"))
}

func getProjectNameForScanTests() string {
	return getProjectNameForTest() + "_for_scan"
}

func getCreateArgs(source string, tags map[string]string, scanTypes string) []string {
	return getCreateArgsWithGroups(source, tags, nil, scanTypes)
}
func getCreateArgsWithGroups(source string, tags map[string]string, groups []string, scanTypes string) []string {
	projectName := getProjectNameForScanTests()
	return getCreateArgsWithNameAndGroups(source, tags, groups, projectName, scanTypes)
}

func getCreateArgsWithName(source string, tags map[string]string, projectName, scanTypes string) []string {
	return getCreateArgsWithNameAndGroups(source, tags, nil, projectName, scanTypes)
}
func getCreateArgsWithNameAndGroups(source string, tags map[string]string, groups []string, projectName, scanTypes string) []string {

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), source,
		flag(params.ScanTypes), scanTypes,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.TagList), formatTags(tags),
		flag(params.BranchFlag), SlowRepoBranch,
		flag(params.ProjectGroupList), formatGroups(groups),
	}

	if strings.Contains(scanTypes, "scs") {
		addSCSDefaultFlagsToArgs(&args)
	}

	return args
}

func executeCreateScan(t *testing.T, args []string, prop ...string) (string, string) {
	var buffer *bytes.Buffer
	if (prop != nil && len(prop) > 0) && prop[0] == "timeout" {
		buffer = executeScanGetBufferWithSpecificTimeout(t, args, 12*time.Minute)
	} else {
		buffer = executeScanGetBuffer(t, args)
	}

	createdScan := wrappers.ScanResponseModel{}
	_ = unmarshall(t, buffer, &createdScan, "Reading scan response JSON should pass")

	assert.Assert(t, createdScan.Status != wrappers.ScanFailed && createdScan.Status != wrappers.ScanCanceled)

	log.Println("Created new project with id: ", createdScan.ProjectID)
	log.Println("Created new scan with id: ", createdScan.ID)

	return createdScan.ID, createdScan.ProjectID
}

func executeScanGetBuffer(t *testing.T, args []string) *bytes.Buffer {
	return executeCmdWithTimeOutNilAssertion(t, "Creating a scan should pass", timeout, args...)
}

func executeScanGetBufferWithSpecificTimeout(t *testing.T, args []string, timeOut time.Duration) *bytes.Buffer {
	return executeCmdWithTimeOutNilAssertion(t, "Creating a scan should pass", timeOut, args...)
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
		"scan", scanList, flag(params.FormatFlag), printer.FormatJSON, flag(params.FilterFlag), scanFilter,
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

// Get a scan workflow and assert it fails
func TestScanWorkflow(t *testing.T) {
	scanID := "fake-scan-id"
	args := []string{
		"scan", "workflow",
		flag(params.ScanIDFlag), scanID,
		flag(params.FormatFlag), printer.FormatJSON,
	}
	cmd := createASTIntegrationTestCommand(t)
	err := execute(cmd, args...)
	assert.Assert(t, err != nil, "Failed showing a scan: response status code 404")
}

func TestScanLogsSAST(t *testing.T) {
	scanID, _ := getRootScan(t)
	args := []string{
		"scan", "logs",
		flag(params.ScanIDFlag), scanID,
		flag(params.ScanTypes), "sast",
	}
	cmd := createASTIntegrationTestCommand(t)
	err := execute(cmd, args...)
	assert.Assert(t, err != nil, "response status code 404")
}

func TestScanLogsKICSDeprecated(t *testing.T) {
	scanID, _ := getRootScan(t)
	args := []string{
		"scan", "logs",
		flag(params.ScanIDFlag), scanID,
		flag(params.ScanTypes), "kics",
	}
	cmd := createASTIntegrationTestCommand(t)
	err := execute(cmd, args...)
	assert.Assert(t, err != nil, "response status code 404")
}

func TestScanLogsKICS(t *testing.T) {
	scanID, _ := getRootScan(t)
	args := []string{
		"scan", "logs",
		flag(params.ScanIDFlag), scanID,
		flag(params.ScanTypes), "iac-security",
	}
	cmd := createASTIntegrationTestCommand(t)
	err := execute(cmd, args...)
	assert.Assert(t, err != nil, "response status code 404")
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
	assertAstError(t, err, "scan completed partially", exitCodes.SastEngineFailedExitCode)
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
		flag(params.IgnorePolicyFlag),
		flag(params.PolicyTimeoutFlag),
		"999999",
	}
	err, _ := executeCommand(t, args...)
	assertAstError(t, err, "scan did not complete successfully", exitCodes.SastEngineFailedExitCode)
}

func assertAstError(t *testing.T, err error, expectedErrorMessage string, expectedExitCode int) {
	var e *wrappers.AstError
	if errors.As(err, &e) {
		assert.Equal(t, e.Error(), expectedErrorMessage)
		assert.Equal(t, e.Code, expectedExitCode)
	} else {
		assertError(t, err, "Error is not of type AstError")
		assert.Assert(t, false, fmt.Sprintf("Error is not of type AstError. Error message: %s", err.Error()))
	}
}

func retrieveResultsFromScanId(t *testing.T, scanId string) (wrappers.ScanResultsCollection, error) {
	resultsArgs := []string{
		"results",
		"show",
		flag(params.ScanIDFlag),
		scanId,
		flag(params.IgnorePolicyFlag),
	}
	executeCmdNilAssertion(t, "Getting results should pass", resultsArgs...)
	file, err := os.ReadFile("cx_result.json")
	defer func() {
		_ = os.Remove("cx_result.json")
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
	args = append(args, flag(params.SastFilterFlag), "!*.java", flag(params.IgnorePolicyFlag))
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

	_ = os.WriteFile(SSHKeyFilePath, []byte(sshKey), 0644)
	defer func() { _ = os.Remove(SSHKeyFilePath) }()

	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), SSHRepo,
		flag(params.BranchFlag), "main",
		flag(params.SSHKeyFlag), SSHKeyFilePath,
		flag(params.ScanTypes), params.IacType,
		flag(params.IgnorePolicyFlag),
	}

	executeCmdWithTimeOutNilAssertion(t, "Create a scan with ssh-key should pass", timeout, args...)
}

func TestScanGLReportValidation(t *testing.T) {
	projectName := getProjectNameForScanTests()
	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), "./data/empty-folder.zip",
		flag(params.ScanTypes), "sast, sca",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.TargetFormatFlag), printer.FormatGLSca,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, err)

	reportFilePath := "./cx_result.gl-sca-report.json"
	schemaURL := "https://gitlab.com/gitlab-org/security-products/security-report-schemas/-/raw/master/dist/dependency-scanning-report-format.json"

	defer os.Remove(reportFilePath)

	schemaLoader := gojsonschema.NewReferenceLoader(schemaURL)
	documentLoader := gojsonschema.NewReferenceLoader("file://" + reportFilePath)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		t.Fatalf("Error occurred during schema validation: %s", err)
	}

	if !result.Valid() {
		for _, desc := range result.Errors() {
			t.Errorf("Validation error: %s", desc)
		}
	}
}

func TestCreateScanFilterZipFile(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), "main",
		flag(params.SourcesFlag), Zip,
		flag(params.SourceDirFilterFlag), "!*.html",
		flag(params.IgnorePolicyFlag),
		flag(params.ScanTypes), params.IacType,
	}

	executeCmdWithTimeOutNilAssertion(t, "Scan must complete successfully", timeout, args...)
}

func TestRunKicsScan(t *testing.T) {
	outputBuffer := executeCmdNilAssertion(
		t, "Runing KICS real-time command should pass",
		scanCommand, kicsRealtimeCommand,
		flag(params.KicsRealtimeFile), fileSourceValueVul,
	)

	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestRunKicsScanWithouResults(t *testing.T) {
	outputBuffer := executeCmdNilAssertion(
		t, "Runing KICS real-time command should pass",
		scanCommand, kicsRealtimeCommand,
		flag(params.KicsRealtimeFile), fileSourceValue,
	)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestRunKicsScanWithoutFileSources(t *testing.T) {
	args := []string{
		scanCommand, kicsRealtimeCommand,
	}
	err, _ := executeCommand(t, args...)
	assertError(t, err, "required flag(s) \"file\" not set")
}

func TestRunKicsScanWithEngine(t *testing.T) {
	outputBuffer := executeCmdNilAssertion(
		t, "Runing KICS real-time with engine command should pass",
		scanCommand, kicsRealtimeCommand,
		flag(params.KicsRealtimeFile), fileSourceValueVul,
		flag(params.KicsRealtimeEngine), engineValue,
	)

	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestRunKicsScanWithInvalidEngine(t *testing.T) {
	args := []string{
		scanCommand, kicsRealtimeCommand,
		flag(params.KicsRealtimeFile), fileSourceValueVul,
		flag(params.KicsRealtimeEngine), invalidEngineValue,
	}
	err, _ := executeCommand(t, args...)
	assertError(t, err,
		util.InvalidEngineMessage)
}

func TestRunKicsScanWithAdditionalParams(t *testing.T) {
	outputBuffer := executeCmdNilAssertion(
		t, "Runing KICS real-time with additional params command should pass",
		scanCommand, kicsRealtimeCommand,
		flag(params.KicsRealtimeFile), fileSourceValueVul,
		flag(params.KicsRealtimeEngine), engineValue,
		flag(params.KicsRealtimeAdditionalParams), additionalParamsValue,
	)

	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestRunScaRealtimeScan(t *testing.T) {
	args := []string{scanCommand, "sca-realtime", "--project-dir", projectDirectory}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)

	// Ensure we have results to read
	err = copyResultsToTempDir()
	assert.NilError(t, err)

	err = realtime.GetSCAVulnerabilities(wrappers.NewHTTPScaRealTimeWrapper())
	assert.NilError(t, err)

	// Run second time to cover SCA Resolver download not needed code
	err, _ = executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestScaRealtimeRequiredAndWrongProjectDir(t *testing.T) {
	args := []string{scanCommand, "sca-realtime"}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, "required flag(s) \"project-dir\" not set", "Sca realtime should fail due missing project directory path")

	args = []string{scanCommand, "sca-realtime", "--project-dir", projectDirectory + "/missingFolder"}

	err, _ = executeCommand(t, args...)
	assert.Error(t, err, "Provided path does not exist: "+projectDirectory+"/missingFolder", "Sca realtime should fail due invalid project directory path")
}

func TestScaRealtimeScaResolverWrongDownloadLink(t *testing.T) {
	err := os.RemoveAll(scaconfig.Params.WorkingDir())
	assert.NilError(t, err)

	args := []string{scanCommand, "sca-realtime", "--project-dir", projectDirectory}

	downloadURL := scaconfig.Params.DownloadURL
	scaconfig.Params.DownloadURL = "https://www.invalid-sca-resolver.com"
	err, _ = executeCommand(t, args...)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(strings.ToLower(err.Error()), strings.ToLower("Invoking HTTP request to download file failed")))

	scaconfig.Params.DownloadURL = downloadURL
	scaconfig.Params.HashDownloadURL = "https://www.invalid-sca-resolver-hash.com"
	err, _ = executeCommand(t, args...)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(strings.ToLower(err.Error()), strings.ToLower("Invoking HTTP request to download file failed")))
}

func copyResultsToTempDir() error {
	// Read all content of src to data, may cause OOM for a large file.
	data, err := os.ReadFile("./data/cx-sca-realtime-results.json")
	if err != nil {
		return err
	}
	// Write data to dst
	scaResolverResultsFileNameDir := filepath.Join(scaconfig.Params.WorkingDir(), realtime.ScaResolverResultsFileName)
	err = os.WriteFile(scaResolverResultsFileNameDir, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func TestScanCreateWithAPIKeyNoTenant(t *testing.T) {
	_ = viper.BindEnv("CX_APIKEY")
	apiKey := viper.GetString("CX_APIKEY")

	outputBuffer := executeCmdNilAssertion(
		t, "Scan create with API key and no tenant should pass",
		scanCommand, "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "./data/sources.zip",
		flag(params.BranchFlag), "main",
		flag(params.AstAPIKeyFlag), apiKey,
		flag(params.ScanTypes), "sast",
		flag(params.DebugFlag),
		flag(params.AsyncFlag),
	)

	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestScanCreateResubmit(t *testing.T) {
	projectName := getProjectNameForScanTests()
	executeCreateScan(t, append(getCreateArgsWithName(Zip, nil, projectName, params.SastType)))
	_, projectID := executeCreateScan(t, append(getCreateArgsWithName(Zip, nil, projectName, ""), flag(params.ScanResubmit)))
	args := []string{
		scanCommand, scanList,
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.FilterFlag), projectIDParams + projectID,
	}
	err, outputBuffer := executeCommand(t, args...)
	scan := []wrappers.ScanResponseModel{}
	_ = unmarshall(t, outputBuffer, &scan, "Reading scan response JSON should pass")
	engines := strings.Join(scan[0].Engines, ",")
	log.Printf("ProjectID for resubmit: %s with engines: %s\n", projectID, engines)
	assert.Assert(t, err == nil && engines == "sast", "")
}

// TestScanTypesValidation must return an error because the user is not allowed to use some scanType
func TestScanTypesValidation(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sast,invalid_scan_type",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "It looks like the")
}

func TestScanTypeApiSecurityWithoutSast(t *testing.T) {
	_, projectName := getRootProject(t)
	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "api-security",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Create a scan should be created only with api security. ")
}

// TestValidateScanTypesUsingInvalidAPIKey error when running a scan with scan-types flag using an invalid api key
func TestValidateScanTypesUsingInvalidAPIKey(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sca,invalid_scan_type",
		flag(params.AstAPIKeyFlag), "invalidAPIKey",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Error validating scan types")
}

func TestScanGeneratingPdfToEmailReport(t *testing.T) {
	_, projectName := getRootProject(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Scan create with API key generating PDF to email report should pass",
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "iac-security",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.TargetFormatFlag), "pdf",
		flag(params.ReportFormatPdfToEmailFlag), "test@checkmarx.com",
	)

	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestScanGeneratingPdfToEmailReportInvalidEmail(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "iac-security",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.TargetFormatFlag), "pdf",
		flag(params.ReportFormatPdfToEmailFlag), "test@checkmarx.com,invalid_email",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Report not sent, invalid email address: invalid_email")
}

func TestScanGeneratingPdfReportWithInvalidPdfOptions(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "iac-security",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.TargetFormatFlag), "pdf",
		flag(params.ReportFormatPdfOptionsFlag), "invalid_option",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "report option \"invalid_option\" unavailable")
}

func TestScanGeneratingPdfReportWithPdfOptions(t *testing.T) {
	_, projectName := getRootProject(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Scan create with API key generating PDF report with options should pass",
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "iac-security",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.TargetFormatFlag), "pdf",
		flag(params.ReportFormatPdfOptionsFlag), "Iac-Security,ScanSummary,ExecutiveSummary,ScanResults",
		flag(params.TargetFlag), fileName,
	)
	defer func() {
		os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
		log.Println("test file removed!")
	}()
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "Report file should exist: "+fileName+printer.FormatPDF)
	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")

}

//func TestScanCreateUsingProjectGroupsAndProjectTags(t *testing.T) {
//	_, projectName := getRootProject(t)
//
//	//outputBuffer := executeCmdNilAssertion(
//	//	t, "Scan create with API key using project groups and project tags should pass",
//	//	scanCommand, "create",
//	//	flag(params.ProjectName), projectName,
//	//	flag(params.SourcesFlag), Zip,
//	//	flag(params.ScanTypes), "sast",
//	//	flag(params.PresetName), "Checkmarx Default",
//	//	flag(params.BranchFlag), "dummy_branch",
//	//	flag(params.ProjectTagList), "integration",
//	//	flag(params.ProjectGroupList), "test",
//	//)
//
//	//assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
//
//}

func TestScanCreate_WhenProjectExists_ShouldNotUpdateGroups(t *testing.T) {
	projectID, projectName := getRootProject(t)
	project := showProject(t, projectID)
	groupsBeforeScanCreate := project.Groups

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sast",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ProjectGroupList), "wrong_group",
		"--async",
	}

	err, _ := executeCommand(t, args...)
	if err != nil {
		assertError(t, err, "running a scan should pass")
	}

	project = showProject(t, projectID)
	groupsAfterScanCreate := project.Groups
	if !reflect.DeepEqual(groupsBeforeScanCreate, groupsAfterScanCreate) {
		t.Errorf("When project exists, groups before and after scan creation should be equal. Got %v, want %v", groupsAfterScanCreate, groupsBeforeScanCreate)
	}

}

func TestScanCreate_WhenProjectExists_ShouldNotUpdateApplication(t *testing.T) {
	projectID, projectName := getRootProject(t)
	project := showProject(t, projectID)
	applicationsBeforeScanCreate := project.ApplicationIds

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sast",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ApplicationName), "wrong_application",
		"--async",
	}

	err, _ := executeCommand(t, args...)
	if err != nil {
		assertError(t, err, "running a scan should pass")
	}

	project = showProject(t, projectID)
	applicationsAfterScanCreate := project.ApplicationIds
	if !reflect.DeepEqual(applicationsBeforeScanCreate, applicationsAfterScanCreate) {
		t.Errorf("When project exists, applications before and after scan creation should be equal. Got %v, want %v", applicationsAfterScanCreate, applicationsBeforeScanCreate)
	}

}
func TestScanCreateExploitablePath(t *testing.T) {
	_, projectName := getRootProject(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Scan create should pass",
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sast",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ProjectTagList), "integration",
		flag(params.ExploitablePathFlag), "true",
		flag(params.LastSastScanTime), "1",
	)

	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}

func TestScanCreateExploitablePathWithoutSAST(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sca",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ProjectTagList), "integration",
		flag(params.ExploitablePathFlag), "true",
		flag(params.LastSastScanTime), "1",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "must enable SAST scan type")
}
func TestScanCreateExploitablePathWithWrongValue(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sca,sast",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ProjectTagList), "integration",
		flag(params.ExploitablePathFlag), "nottrueorfalse",
		flag(params.LastSastScanTime), "1",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Invalid value for --sca-exploitable-path flag")
}

func TestScanCreateLastSastScanTimeWithInvalidValue(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sca,sast",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ProjectTagList), "integration",
		flag(params.ExploitablePathFlag), "false",
		flag(params.LastSastScanTime), "notanumber",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Invalid value for --sca-last-sast-scan-time flag")
}

func TestCreateScanProjectPrivatePackage(t *testing.T) {
	_, projectName := getRootProject(t)

	outputBuffer := executeCmdNilAssertion(
		t, "Scan create should pass",
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "kics",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ProjectTagList), "integration",
		flag(params.ProjecPrivatePackageFlag), "true",
	)

	assert.Assert(t, outputBuffer != nil, "Scan must complete successfully")
}
func TestCreateScanProjectPrivatePackageWithInvalidValue(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "kics",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ProjectTagList), "integration",
		flag(params.ProjecPrivatePackageFlag), "nottrueorfalse",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Invalid value for --project-private-package flag")
}

func TestCreateScanSBOMReportFormatWithoutSCA(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "kics",
		flag(params.PresetName), "Checkmarx Default",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ProjectTagList), "integration",
		flag(params.TargetFormatFlag), "sbom",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "SCA engine must be enabled on scan summary")
}

func TestScanWithPolicy(t *testing.T) {
	args := []string{scanCommand, "create",
		flag(params.ProjectName), "TiagoBaptista/testingCli/testingCli",
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "main",
		flag(params.TargetFormatFlag), "markdown,summaryConsole,summaryHTML"}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestScanWithPolicyTimeout(t *testing.T) {
	args := []string{scanCommand, "create",
		flag(params.ProjectName), "TiagoBaptista/testingCli/testingCli",
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sast",
		flag(params.BranchFlag), "main",
		flag(params.PolicyTimeoutFlag), "-1"}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, "--policy-timeout should be equal or higher than 0")
}

func TestCreateScan_WithTypeScs_Success(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "scs",
		flag(params.BranchFlag), "main",
		flag(params.SCSRepoURLFlag), scsRepoURL,
		flag(params.SCSRepoTokenFlag), scsRepoToken,
		flag(params.TargetFormatFlag), strings.Join(
			[]string{
				printer.FormatJSON,
				printer.FormatSarif,
				printer.FormatSonar,
				printer.FormatSummaryConsole,
				printer.FormatSummaryJSON,
				printer.FormatPDF,
				printer.FormatSummaryMarkdown,
			}, ",",
		),
	}

	executeCmdWithTimeOutNilAssertion(t, "SCS scan must complete successfully", 4*time.Minute, args...)
}

func TestCreateScan_WithTypeScsAndOnlyScorecard_Success(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "scs",
		flag(params.BranchFlag), "main",
		flag(params.SCSRepoURLFlag), scsRepoURL,
		flag(params.SCSRepoTokenFlag), scsRepoToken,
		flag(params.SCSEnginesFlag), commands.ScsScoreCardType,
		flag(params.TargetFormatFlag), strings.Join(
			[]string{
				printer.FormatJSON,
				printer.FormatSarif,
				printer.FormatSonar,
				printer.FormatSummaryConsole,
				printer.FormatSummaryJSON,
				printer.FormatPDF,
				printer.FormatSummaryMarkdown,
			}, ",",
		),
	}

	executeCmdWithTimeOutNilAssertion(t, "SCS scan with only Scorecard must complete successfully", 4*time.Minute, args...)
}

func TestCreateScan_WithNoScanTypesAndScsFlagsNotPresent_SuccessAndScsScanned(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.BranchFlag), "main",
	}

	output := executeCmdWithTimeOutNilAssertion(t, "Scan must complete successfully if no scan-types specified and with missing scs-repo flags", timeout, args...)
	assert.Assert(t, strings.Contains(output.String(), params.ScsType), "SCS scan should run")
}

func TestCreateScan_WithNoScanTypesFlagButScsFlagsPresent_SuccessAndScsScanned(t *testing.T) {
	_, projectName := getRootProject(t)
	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.BranchFlag), "main",
		flag(params.SCSRepoURLFlag), scsRepoURL,
		flag(params.SCSRepoTokenFlag), scsRepoToken,
	}

	output := executeCmdWithTimeOutNilAssertion(t, "Scan must complete successfully if no scan-types specified and scs-repo flags are present", 4*time.Minute, args...)
	assert.Assert(t, strings.Contains(output.String(), params.ScsType), "Scs scan should run if all required flags are provided")
}

func TestCreateScan_WithTypeScsMissingRepoURL_Fail(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "iac-security, scs",
		flag(params.BranchFlag), "main",
		flag(params.SCSRepoTokenFlag), scsRepoToken,
	}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, commands.ScsRepoRequiredMsg)
}

func TestCreateScan_WithTypeScsMissingRepoToken_Fail(t *testing.T) {
	_, projectName := getRootProject(t)
	scsRepoTokenEnvValue := os.Getenv(params.ScsRepoTokenEnv)
	defer setEnvVars(map[string]string{params.ScsRepoTokenEnv: scsRepoTokenEnvValue})

	setEnvVars(map[string]string{
		params.ScsRepoTokenEnv: "",
	})

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "iac-security, scs",
		flag(params.BranchFlag), "main",
		flag(params.SCSRepoURLFlag), scsRepoURL,
	}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, commands.ScsRepoRequiredMsg)
}

func TestCreateScan_ScsRepoTokenEnvConfigured_Success(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "iac-security, scs",
		flag(params.BranchFlag), "main",
		flag(params.SCSRepoURLFlag), scsRepoURL,
	}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, commands.ScsRepoRequiredMsg)
}

func TestCreateScan_WithTypeScsOnlySecretDetection_Success(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "scs",
		flag(params.BranchFlag), "main",
		flag(params.SCSEnginesFlag), commands.ScsSecretDetectionType,
	}

	executeCmdWithTimeOutNilAssertion(t,
		"SCS with only secret-detection scan must complete successfully, even if missing scs-repo flags", timeout, args...)
}

func TestCreateScan_WithNoScanTypesFlagScsAndOnlySecretDetection_Success(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.BranchFlag), "main",
		flag(params.SCSEnginesFlag), commands.ScsSecretDetectionType,
	}

	executeCmdWithTimeOutNilAssertion(t,
		"SCS with only secret-detection scan must complete successfully, even if missing scs-repo flags", timeout, args...)
}

func TestCreateScan_WithScanTypesScsAndOnlyScorecardMissingRepoFlags_Fail(t *testing.T) {
	_, projectName := getRootProject(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "scs",
		flag(params.BranchFlag), "main",
		flag(params.SCSEnginesFlag), commands.ScsScoreCardType,
	}

	err, _ := executeCommand(t, args...)
	assert.Error(t, err, commands.ScsRepoRequiredMsg)
}

func TestScanListWithFilters(t *testing.T) {
	args := []string{
		"scan", "list",
		flag(params.FilterFlag), "limit=100",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "")
}

func TestScanListWithBigLimitAndOtherFilters(t *testing.T) {
	args := []string{
		"scan", "list",
		flag(params.FilterFlag), "limit=10000,project-id=6cd7afbd-3d21-44b9-a72f-8a7eb351b5a5,branch=develop",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "")
}

func TestScanListWithBigLimit(t *testing.T) {
	args := []string{
		"scan", "list",
		flag(params.FilterFlag), "limit=10000",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "")
}

func addSCSDefaultFlagsToArgs(args *[]string) {
	*args = append(*args, flag(params.SCSRepoURLFlag), scsRepoURL, flag(params.SCSRepoTokenFlag), scsRepoToken)
}

func TestCreateScanAndValidateCheckmarxDomains(t *testing.T) {
	wrappers.Domains = make(map[string]struct{})
	_, _ = executeCreateScan(t, getCreateArgsWithGroups(Zip, Tags, Groups, "iac-security"))
	baseUrl, _ := wrappers.GetURL("", "")
	authUri, _ := wrappers.GetAuthURI()
	usedDomainsFromConfig := []string{baseUrl, authUri}
	validateCheckmarxDomains(t, usedDomainsFromConfig)
}

func validateCheckmarxDomains(t *testing.T, usedDomainsInTests []string) {
	usedDomains := wrappers.Domains
	for domain, _ := range usedDomains {
		assert.Assert(t, utils.Contains(usedDomainsInTests, domain), "Domain "+domain+" not found in used domains")
	}
}

func TestCreateScan_TwoScansWithSameBranchNameWithWhiteSpace_Success(t *testing.T) {
	projectName := GenerateRandomProjectNameForScan()
	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "iac-security",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	_, projectID := executeCreateScan(t, args)
	args2 := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "iac-security",
		flag(params.BranchFlag), "   dummy_branch    ",
	}
	err, _ := executeCommand(t, args2...)
	assert.NilError(t, err)

	response := listScanByProjectID(t, projectID)
	assert.Assert(t, len(response) == 2)
	for _, scan := range response {
		assert.Equal(t, scan.Branch, "dummy_branch", "Branch name should be dummy_branch")
	}
}
func listScanByProjectID(t *testing.T, projectID string) []wrappers.ScanResponseModel {
	scanFilter := fmt.Sprintf("project-id=%s", projectID)
	outputBuffer := executeCmdNilAssertion(
		t,
		"Getting the scan should pass",
		"scan", scanList, flag(params.FormatFlag), printer.FormatJSON, flag(params.FilterFlag), scanFilter,
	)
	// Read response from buffer
	var scanList []wrappers.ScanResponseModel
	_ = unmarshall(t, outputBuffer, &scanList, "Reading scan response JSON should pass")
	return scanList
}

func TestCreateAsyncScan_CallExportServiceBeforeScanFinishWithRetry_Success(t *testing.T) {
	createASTIntegrationTestCommand(t)
	configuration.LoadConfiguration()
	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.ScanTypes), "sca",
		flag(params.BranchFlag), "main",
		flag(params.AsyncFlag),
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, _ := executeCreateScan(t, args)
	exportRes, err := services.GetExportPackage(wrappers.NewExportHTTPWrapper("api/sca/export"), scanID, false)
	asserts.Nil(t, err)
	assert.Assert(t, exportRes != nil, "Export response should not be nil")
}

func TestCreateScanWithResubmitFlag_ProjectNotExist_ScanCreatedSuccessfullyWithDefaultConfig(t *testing.T) {
	projectName := GenerateRandomProjectNameForScan()
	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), Zip,
		flag(params.BranchFlag), "main",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.ScanResubmit),
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}
func TestScanCreate_WithContainerFilterFlags_CreatingScanSuccessfully(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	var createdScan wrappers.ScanResponseModel
	var createdScanConfig wrappers.Config
	scansPath := viper.GetString(params.ScansPathKey)
	scanWrapper := wrappers.NewHTTPScansWrapper(scansPath)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), GenerateRandomProjectNameForScan(),
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ContainersFileFolderFilterFlag), "!*.log",
		flag(params.ContainersExcludeNonFinalStagesFlag),
		flag(params.ContainersImageTagFilterFlag), "*dev",
		flag(params.ContainersPackageFilterFlag), "^internal-.*",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.AsyncFlag),
	}
	scanID, projectID := executeCreateScan(t, args)

	mapParams := make(map[string]string)
	mapParams["project-id"] = projectID
	allScansModel, _, _ := scanWrapper.Get(mapParams)

	createdScan = allScansModel.Scans[0]

	assert.Assert(t, createdScan.ID == scanID, "Scan ID should be equal")
	assert.Equal(t, len(createdScan.Metadata.Configs), 1, "Scan should have only containers config")

	createdScanConfig = createdScan.Metadata.Configs[0]

	assert.Equal(t, createdScanConfig.Type, params.ContainersType, "Scan type should be equal")
	assert.Equal(t, createdScanConfig.Value[commands.ConfigContainersFilesFilterKey], "!*.log", "File/Folder filter should be equal")
	assert.Equal(t, createdScanConfig.Value[commands.ConfigContainersNonFinalStagesFilterKey], "true", "Exclude non final stages should be equal")
	assert.Equal(t, createdScanConfig.Value[commands.ConfigContainersImagesFilterKey], "*dev", "Image tag filter should be equal")
	assert.Equal(t, createdScanConfig.Value[commands.ConfigContainersPackagesFilterKey], "^internal-.*", "Package filter should be equal")

}

func TestScanCreate_WithContainerFilterFlagsAndResubmitFlag_CreatingScanWithLatestScanConfigurationSuccessfully(t *testing.T) {
	bindKeysToEnvAndDefault(t)

	var createdScan wrappers.ScanResponseModel
	var createdScanConfig wrappers.Config

	scansPath := viper.GetString(params.ScansPathKey)
	scanWrapper := wrappers.NewHTTPScansWrapper(scansPath)

	projectName := GenerateRandomProjectNameForScan()

	args := []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ContainersFileFolderFilterFlag), "!*.log",
		flag(params.ContainersImageTagFilterFlag), "*dev",
		flag(params.ContainersPackageFilterFlag), "^internal-.*",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.AsyncFlag),
	}

	_, projectID := executeCreateScan(t, args)

	args = []string{
		"scan", "create",
		flag(params.ProjectName), projectName,
		flag(params.SourcesFlag), ".",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ContainersFileFolderFilterFlag), "!dockerfile",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		flag(params.AsyncFlag),
		flag(params.ScanResubmit),
	}

	resubmitScanID, resubmitProjectID := executeCreateScan(t, args)

	assert.Equal(t, projectID, resubmitProjectID, "Project ID should be equal")

	mapParams := make(map[string]string)
	mapParams["project-id"] = resubmitProjectID
	allScansModel, _, _ := scanWrapper.Get(mapParams)

	createdScan = allScansModel.Scans[0]

	assert.Assert(t, createdScan.ID == resubmitScanID, "Scan ID should be equal")
	assert.Equal(t, len(createdScan.Metadata.Configs), 1, "Scan should have only containers config")

	createdScanConfig = createdScan.Metadata.Configs[0]

	assert.Equal(t, createdScanConfig.Type, params.ContainersType, "Scan type should be equal")
	assert.Equal(t, createdScanConfig.Value[commands.ConfigContainersFilesFilterKey], "!dockerfile", "File/Folder filter should be equal")
	assert.Equal(t, createdScanConfig.Value[commands.ConfigContainersNonFinalStagesFilterKey], nil, "Exclude non final stages should be equal")
	assert.Equal(t, createdScanConfig.Value[commands.ConfigContainersImagesFilterKey], "*dev", "Image tag filter should be equal")
	assert.Equal(t, createdScanConfig.Value[commands.ConfigContainersPackagesFilterKey], "^internal-.*", "Package filter should be equal")
}

func TestCreateScanWithAsyncFlag_TryShowResults_PolicyNotEvaluated(t *testing.T) {
	createASTIntegrationTestCommand(t)
	configuration.LoadConfiguration()
	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), Zip,
		flag(params.ScanTypes), "sast,iac-security,sca",
		flag(params.BranchFlag), "main",
		flag(params.AsyncFlag),
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, _ := executeCreateScan(t, args)
	assert.Assert(t, scanID != "", "Scan ID should not be empty")

	var buf bytes.Buffer
	log.SetOutput(&buf)

	_ = executeCmdNilAssertion(
		t, "Results show generating JSON report with options should pass",
		"results", "show",
		flag(params.ScanIDFlag), scanID,
		flag(params.TargetFormatFlag), printer.FormatSummaryConsole,
		flag(params.DebugFlag),
	)
	log.SetOutput(os.Stderr)
	assert.Assert(t, strings.Contains(buf.String(), "Policy violations aren't returned in the pipeline for scans run in async mode."), "policy shouldn't evaluate in running scan")
}
