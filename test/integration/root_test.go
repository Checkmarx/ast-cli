//go:build integration

package integration

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

const (
	FullScanWait          = 180
	ScanPollSleep         = 5
	Dir                   = "./data"
	Zip                   = "data/sources.zip"
	SlowRepo              = "https://github.com/WebGoat/WebGoat"
	SSHRepo               = "git@github.com:pedrompflopes/ast-jenkins-docker.git"
	SlowRepoBranch        = "develop"
	resolverEnvVar        = "SCA_RESOLVER"
	resolverEnvVarDefault = "./ScaResolver"
)

var Tags = map[string]string{
	"Galactica":   "",
	"Integration": "Tests",
}

var Groups = []string{
	"it_test_group_1",
	"it_test_group_2",
}

var testInstance *testing.T
var rootScanId string
var rootEnginesScanId string
var rootScanProjectId string
var rootEnginesScanProjectId string
var rootProjectId string
var rootProjectName string

func TestMain(m *testing.M) {
	log.Println("CLI integration tests started")
	viper.SetDefault(resolverEnvVar, resolverEnvVarDefault)
	exitVal := m.Run()
	//deleteScanAndProject()
	log.Println("CLI integration tests done")
	os.Exit(exitVal)
}

func TestRootVersion(t *testing.T) {
	testInstance = t
	executeCmdNilAssertion(t, "test root version", "version")
}

// Create or return a scan to be shared between tests
func getRootScan(t *testing.T, scanTypes ...string) (string, string) {
	testInstance = t

	if len(rootScanId) > 0 {
		log.Println("Using the scanID: ", rootScanId)
		log.Println("Using the projectID: ", rootScanProjectId)
		return rootScanId, rootScanProjectId
	}
	if len(scanTypes) == 0 {
		rootScanId, rootScanProjectId = createScan(testInstance, Zip, Tags)
		return rootScanId, rootScanProjectId
	} else {
		rootEnginesScanId, rootEnginesScanProjectId = createScanWithEngines(testInstance, Zip, Tags, strings.Join(scanTypes, ","))
		return rootEnginesScanId, rootEnginesScanProjectId
	}
}

// Delete scan and projects
func deleteScanAndProject() {
	if len(rootScanId) > 0 {
		deleteScan(testInstance, rootScanId)
		rootScanId = ""
	}
	if len(rootScanProjectId) > 0 {
		deleteProject(testInstance, rootScanProjectId)
		rootScanProjectId = ""
	}
	if len(rootProjectId) > 0 {
		deleteProject(testInstance, rootProjectId)
		rootProjectId = ""
	}
}

// Create or return a project to be shared between tests
func getRootProject(t *testing.T) (string, string) {
	testInstance = t

	if len(rootProjectId) > 0 {
		fmt.Printf("Using the projectID: %s", rootProjectId)
		log.Println("Using the projectID: ", rootProjectId)
		log.Println("Using the projectName: ", rootProjectName)
		return rootProjectId, rootProjectName
	}

	rootProjectId, rootProjectName = createProject(t, Tags, Groups)

	//--------------------Write project name to file to delete it later--------------------
	_ = WriteProjectNameToFile(fmt.Sprint(getProjectNameForTest(), "_for_scan"))
	_ = WriteProjectNameToFile(rootProjectName)
	//-------------------------------------------------------------------------------------

	return rootProjectId, rootProjectName
}

func isFFEnabled(t *testing.T, featureFlag string) bool {
	// createASTIntegrationTestCommand is called just to load the FF values
	createASTIntegrationTestCommand(t)

	featureFlagsPath := viper.GetString(commonParams.FeatureFlagsKey)
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)

	flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, featureFlag)
	return flagResponse.Status
}

func TestSetLogOutputFromFlag_InvalidDir(t *testing.T) {
	err, _ := executeCommand(t, "auth", "validate", "--log-file", "/custom/path")
	assert.ErrorContains(t, err, "The specified directory path does not exist.")
}

func TestSetLogOutputFromFlag_EmptyDirPath(t *testing.T) {
	err, _ := executeCommand(t, "auth", "validate", "--log-file", "")
	assert.ErrorContains(t, err, "flag needs an argument")
}

func TestSetLogOutputFromFlag_DirPathIsFilePath(t *testing.T) {
	tempFile, err := os.CreateTemp("", "ast-cli.txt")
	defer func(path string) {
		_ = os.Remove(path)
	}(tempFile.Name())
	err, _ = executeCommand(t, "auth", "validate", "--log-file", tempFile.Name())
	assert.ErrorContains(t, err, "Expected a directory path but got a file")
}

func TestSetLogOutputFromFlag_DirPathPermissionDenied(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tempdir")
	_ = os.Chmod(tempDir, 0000)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)
	err, _ = executeCommand(t, "auth", "validate", "--log-file", tempDir)
	assert.ErrorContains(t, err, "Permission denied: cannot write to directory")
}

func TestSetLogOutputFromFlag_DirPath_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tempdir")
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)
	err, _ = executeCommand(t, "auth", "validate", "--log-file", tempDir)
	assert.NilError(t, err)
}

func TestSetLogOutputFromFlag_DirPath_Console_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tempdir")
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)
	err, _ = executeCommand(t, "auth", "validate", "--log-file-console", tempDir)
	assert.NilError(t, err)
}
