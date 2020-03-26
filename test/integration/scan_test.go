// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	scansRESTApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	"github.com/spf13/viper"
	"gotest.tools/assert/cmp"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestRunScanSourcesFile(t *testing.T) {
	viper.SetDefault("TEST_FULL_SCAN_WAIT_COMPLETED_SECONDS", "210")
	fullScanWaitTime := viper.GetInt("TEST_FULL_SCAN_WAIT_COMPLETED_SECONDS")
	viper.SetDefault("TEST_INC_SCAN_WAIT_COMPLETED_SECONDS", "120")
	incScanWaitTime := viper.GetInt("TEST_INC_SCAN_WAIT_COMPLETED_SECONDS")

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
	log.Printf("Waiting %d seconds for the full scan to complete...\n", fullScanWaitTime)
	// Wait for the scan to finish. See it's completed successfully
	time.Sleep(time.Duration(fullScanWaitTime) * time.Second)

	getBuffer := bytes.NewBufferString("")
	getCommand := createASTIntegrationTestCommand()
	getCommand.SetOut(getBuffer)
	err = execute(getCommand, "-v", "scan", "get", createdScan.ID)
	assert.NilError(t, err)
	// Read response from buffer
	var getScanJSON []byte
	getScanJSON, err = ioutil.ReadAll(getBuffer)
	assert.NilError(t, err, "Reading scan response JSON should pass")
	getScan := scansRESTApi.ScanResponseModel{}
	err = json.Unmarshal(getScanJSON, &getScan)
	assert.NilError(t, err, "Parsing scan response JSON should pass")
	assert.Assert(t, cmp.Equal(getScan.ID, createdScan.ID))
	assert.Assert(t, getScan.Status == scansRESTApi.ScanCompleted)

	// TODO get results

	// Create an incremental scan
	incBuff := bytes.NewBufferString("")
	createIncCommand := createASTIntegrationTestCommand()
	createIncCommand.SetOut(incBuff)
	err = execute(createIncCommand, "-v", "scan", "create", "--inputFile", "scan_inc_payload.json", "--sources", "sources_inc.zip")
	assert.NilError(t, err, "Creating an incremental scan should pass")
	// Read response from buffer
	var createdIncScanJSON []byte
	createdIncScanJSON, err = ioutil.ReadAll(incBuff)
	assert.NilError(t, err, "Reading incremental scan response JSON should pass")
	createdIncScan := scansRESTApi.ScanResponseModel{}
	err = json.Unmarshal(createdIncScanJSON, &createdIncScan)
	assert.NilError(t, err, "Parsing incremental scan response JSON should pass")
	assert.Assert(t, createdIncScan.Status == scansRESTApi.ScanCreated)

	log.Printf("Waiting %d seconds for the incremental scan to complete...\n", incScanWaitTime)
	// Wait for the inc scan to finish. See it's completed successfully
	time.Sleep(time.Duration(incScanWaitTime) * time.Second)

	getIncBuffer := bytes.NewBufferString("")
	getIncCommand := createASTIntegrationTestCommand()
	getIncCommand.SetOut(getIncBuffer)
	err = execute(getIncCommand, "-v", "scan", "get", createdIncScan.ID)
	assert.NilError(t, err)
	// Read response from buffer
	var getIncScanJSON []byte
	getIncScanJSON, err = ioutil.ReadAll(getIncBuffer)
	assert.NilError(t, err, "Reading incremental scan response JSON should pass")
	getIncScan := scansRESTApi.ScanResponseModel{}
	err = json.Unmarshal(getIncScanJSON, &getIncScan)
	assert.NilError(t, err, "Parsing incremental scan response JSON should pass")
	assert.Assert(t, cmp.Equal(getIncScan.ID, createdIncScan.ID))
	assert.Assert(t, getIncScan.Status == scansRESTApi.ScanCompleted)

	// TODO get inc results
	// TODO Compare numbers
}
