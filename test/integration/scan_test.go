// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"
	"time"

	scansRESTApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	"gotest.tools/assert/cmp"

	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestScansE2E(t *testing.T) {
	viper.SetDefault("TEST_FULL_SCAN_WAIT_COMPLETED_SECONDS", "120")
	fullScanWaitTime := viper.GetInt("TEST_FULL_SCAN_WAIT_COMPLETED_SECONDS")
	viper.SetDefault("TEST_INC_SCAN_WAIT_COMPLETED_SECONDS", "60")
	incScanWaitTime := viper.GetInt("TEST_INC_SCAN_WAIT_COMPLETED_SECONDS")

	scanID := createScanSourcesFile(t)
	log.Printf("Waiting %d seconds for the full scan to complete...\n", fullScanWaitTime)
	// Wait for the scan to finish. See it's completed successfully
	time.Sleep(time.Duration(fullScanWaitTime) * time.Second)
	getScanByID(t, scanID, scansRESTApi.ScanCompleted)
	incScanID := createIncScan(t)
	log.Printf("Waiting %d seconds for the incremental scan to complete...\n", incScanWaitTime)
	// Wait for the inc scan to finish. See it's completed successfully
	time.Sleep(time.Duration(incScanWaitTime) * time.Second)
	getScanByID(t, incScanID, scansRESTApi.ScanCompleted)
	getAllScans(t, scanID, incScanID)
	getScansTags(t)
}

func createScanSourcesFile(t *testing.T) string {
	// Create a full scan
	b := bytes.NewBufferString("")
	createCommand := createASTIntegrationTestCommand()
	createCommand.SetOut(b)
	err := execute(createCommand, "-v", "scan", "create", "--inputFile", "scan_payload.json", "--sources", "sources.zip")
	assert.NilError(t, err, "Creating a scan should pass")
	// Read response from buffer
	var createdScanJSON []byte
	createdScanJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading scan response JSON should pass")
	createdScan := scansRESTApi.ScanResponseModel{}
	err = json.Unmarshal(createdScanJSON, &createdScan)
	assert.NilError(t, err, "Parsing scan response JSON should pass")
	assert.Assert(t, createdScan.Status == scansRESTApi.ScanCreated)
	log.Printf("Scan ID %s created in test", createdScan.ID)
	return createdScan.ID
}

func deleteScan(t *testing.T) {

}
func getAllScans(t *testing.T, scanID, incScanID string) {
	b := bytes.NewBufferString("")
	getAllCommand := createASTIntegrationTestCommand()
	getAllCommand.SetOut(b)
	err := execute(getAllCommand, "-v", "scan", "get-all")
	assert.NilError(t, err, "Getting all scans should pass")
	// Read response from buffer
	var getAllJSON []byte
	getAllJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading all scans response JSON should pass")
	allScans := scansRESTApi.SlicedScansResponseModel{}
	err = json.Unmarshal(getAllJSON, &allScans)
	assert.NilError(t, err, "Parsing all scans response JSON should pass")
	assert.Assert(t, allScans.Limit == 20, "Limit should be 20")
	assert.Assert(t, allScans.Offset == 0, "Offset should be 0")
	assert.Assert(t, allScans.TotalCount == 2, "Total should be 2")
	assert.Assert(t, len(allScans.Scans) == 2, "Total should be 2")
	assert.Assert(t, allScans.Scans[0].ID == incScanID)
	assert.Assert(t, allScans.Scans[1].ID == scanID)
}
func getScanByID(t *testing.T, scanID string, status string) {
	getBuffer := bytes.NewBufferString("")
	getCommand := createASTIntegrationTestCommand()
	getCommand.SetOut(getBuffer)
	err := execute(getCommand, "-v", "scan", "get", scanID)
	assert.NilError(t, err)
	// Read response from buffer
	var getScanJSON []byte
	getScanJSON, err = ioutil.ReadAll(getBuffer)
	assert.NilError(t, err, "Reading scan response JSON should pass")
	getScan := scansRESTApi.ScanResponseModel{}
	err = json.Unmarshal(getScanJSON, &getScan)
	assert.NilError(t, err, "Parsing scan response JSON should pass")
	assert.Assert(t, cmp.Equal(getScan.ID, scanID))
	assert.Assert(t, string(getScan.Status) == status)
}

func getScansTags(t *testing.T) {
	b := bytes.NewBufferString("")
	tagsCommand := createASTIntegrationTestCommand()
	tagsCommand.SetOut(b)
	err := execute(tagsCommand, "-v", "scan", "tags")
	assert.NilError(t, err, "Getting tags should pass")
	// Read response from buffer
	var tagsJSON []byte
	tagsJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading tags JSON should pass")
	tags := []string{}
	err = json.Unmarshal(tagsJSON, &tags)
	assert.NilError(t, err, "Parsing tags JSON should pass")
	assert.Assert(t, tags != nil)
	assert.Assert(t, len(tags) == 4)
}

func createIncScan(t *testing.T) string {
	// Create an incremental scan
	incBuff := bytes.NewBufferString("")
	createIncCommand := createASTIntegrationTestCommand()
	createIncCommand.SetOut(incBuff)
	err := execute(createIncCommand, "-v", "scan", "create", "--inputFile", "scan_inc_payload.json", "--sources", "sources_inc.zip")
	assert.NilError(t, err, "Creating an incremental scan should pass")
	// Read response from buffer
	var createdIncScanJSON []byte
	createdIncScanJSON, err = ioutil.ReadAll(incBuff)
	assert.NilError(t, err, "Reading incremental scan response JSON should pass")
	createdIncScan := scansRESTApi.ScanResponseModel{}
	err = json.Unmarshal(createdIncScanJSON, &createdIncScan)
	assert.NilError(t, err, "Parsing incremental scan response JSON should pass")
	assert.Assert(t, createdIncScan.Status == scansRESTApi.ScanCreated)
	return createdIncScan.ID
}
