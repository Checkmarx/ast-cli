//go:build integration

package integration

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/spf13/viper"
)

const (
	FullScanWait          = 60
	ScanPollSleep         = 5
	Dir                   = "./data"
	Zip                   = "data/sources.zip"
	SlowRepo              = "https://github.com/WebGoat/WebGoat"
	SSHRepo               = "git@github.com:hmmachadocx/hmmachado_dummy_project.git"
	SlowRepoBranch        = "develop"
	resolverEnvVar        = "SCA_RESOLVER"
	resolverEnvVarDefault = "./ScaResolver"
)

var Tags = map[string]string{
	"Galactica":   "",
	"Integration": "Tests",
}

var testInstance *testing.T
var rootScanId string
var rootScanProjectId string
var rootProjectId string
var rootProjectName string

func TestMain(m *testing.M) {
	log.Println("CLI integration tests started")
	viper.SetDefault(resolverEnvVar, resolverEnvVarDefault)
	exitVal := m.Run()
	deleteScanAndProject()
	log.Println("CLI integration tests done")
	os.Exit(exitVal)
}

// Create or return a scan to be shared between tests
func getRootScan(t *testing.T) (string, string) {
	testInstance = t

	if len(rootScanId) > 0 {
		log.Println("Using the scanID: ", rootScanId)
		log.Println("Using the projectID: ", rootScanProjectId)
		return rootScanId, rootScanProjectId
	}

	rootScanId, rootScanProjectId = createScan(testInstance, Zip, Tags)

	return rootScanId, rootScanProjectId
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

	rootProjectId, rootProjectName = createProject(t, Tags)

	return rootProjectId, rootProjectName
}
