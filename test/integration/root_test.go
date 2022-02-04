//go:build integration

package integration

import (
	"log"
	"os"
	"testing"
)

const (
	FullScanWait   = 60
	ScanPollSleep  = 5
	Dir            = "./data"
	Zip            = "data/sources.zip"
	SlowRepo       = "https://github.com/WebGoat/WebGoat"
	SlowRepoBranch = "develop"
)

var Tags = map[string]string{
	"it_test_tag_1": "",
	"it_test_tag_2": "val",
	"it_test_tag_3": "",
}

var testInstance *testing.T
var rootScanId string
var rootScanProjectId string
var rootProjectId string
var rootProjectName string

func TestMain(m *testing.M) {
	log.Println("CLI integration tests started")
	exitVal := m.Run()
	deleteScanAndProject()
	log.Println("CLI integration tests done")
	os.Exit(exitVal)
}

// Create or return a scan to be shared between tests
func getRootScan(t *testing.T) (string, string) {
	testInstance = t

	if len(rootScanId) > 0 {
		return rootScanId, rootScanProjectId
	}

	rootScanId, rootScanProjectId = createScan(testInstance, Zip, Tags)

	return rootScanId, rootScanProjectId
}

// Delete scan and projects
func deleteScanAndProject() {
	if len(rootScanId) > 0 {
		deleteScan(testInstance, rootScanId)
	}
	if len(rootScanProjectId) > 0 {
		deleteProject(testInstance, rootScanProjectId)
	}
	if len(rootProjectId) > 0 {
		deleteProject(testInstance, rootProjectId)
	}
}

// Create or return a project to be shared between tests
func getRootProject(t *testing.T) (string, string) {
	testInstance = t

	if len(rootProjectId) > 0 {
		return rootProjectId, rootProjectName
	}

	rootProjectId, rootProjectName = createProject(t, Tags)

	return rootProjectId, rootProjectName
}
