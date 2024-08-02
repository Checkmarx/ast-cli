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
		fmt.Printf("Using the projectID: " + rootProjectId)
		log.Println("Using the projectID: ", rootProjectId)
		log.Println("Using the projectName: ", rootProjectName)
		return rootProjectId, rootProjectName
	}

	rootProjectId, rootProjectName = createProject(t, Tags, Groups)

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
