//go:build integration

package integration

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

const (
	FullScanWait          = 180
	ScanPollSleep         = 5
	Dir                   = "./data"
	Zip                   = "data/sources.zip"
	SlowRepo              = "https://github.com/WebGoat/WebGoat"
	SSHRepo               = "git@github.com:hmmachadocx/hmmachado_dummy_project.git"
	SlowRepoBranch        = "develop"
	resolverEnvVar        = "SCA_RESOLVER"
	resolverEnvVarDefault = "./ScaResolver"
	outputDir             = "C:\\temp"
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

	filePath := filepath.Join(outputDir, "test_durations.txt")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		os.Exit(1)
	}
	defer file.Close()
	log.Println("File path of test_durations is: ", filePath)

	start := time.Now()
	exitVal := m.Run()
	duration := time.Since(start)

	_, err = file.WriteString(fmt.Sprintf("Total time: %s\n", duration))
	if err != nil {
		fmt.Println("Failed to write to file:", err)
	}

	log.Println("CLI integration tests done. Durations file saved at: ", filePath)
	os.Exit(exitVal)
}

func recordDuration(t *testing.T, name string, start time.Time) {
	duration := time.Since(start)
	filePath := filepath.Join(outputDir, "test_durations.txt")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%s = %s\n", name, duration))
	if err != nil {
		t.Fatalf("Failed to write to file: %v", err)
	}
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
