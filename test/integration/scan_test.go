package integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	scansRESTApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	"gotest.tools/assert/cmp"

	"gotest.tools/assert"
)

// Create a scan. Get it's Id. Keep polling and see it's completed. Then run incremental scan
// Get the tags from the scan

func TestRunScanSourcesFile(t *testing.T) {
	// Create a full scan and get it's ID
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
}
