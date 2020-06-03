// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"testing"
	"time"

	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/v1/rest"
	"gotest.tools/assert/cmp"

	"github.com/spf13/viper"
	"gotest.tools/assert"
)

const (
	scanResultsNum    = 387
	incScanResultsNum = 409
)

func TestScansE2E(t *testing.T) {
	waitForResults := 10
	viper.SetDefault("TEST_FULL_SCAN_WAIT_COMPLETED_SECONDS", "120")
	fullScanWaitTime := viper.GetInt("TEST_FULL_SCAN_WAIT_COMPLETED_SECONDS")
	viper.SetDefault("TEST_INC_SCAN_WAIT_COMPLETED_SECONDS", "60")
	incScanWaitTime := viper.GetInt("TEST_INC_SCAN_WAIT_COMPLETED_SECONDS")

	scanID := createScanSourcesFile(t)
	log.Printf("Waiting %d seconds for the full scan to complete...\n", fullScanWaitTime)
	// Wait for the scan to finish. See it's completed successfully
	scanCompletedCh := make(chan bool, 1)
	pollScanUntilStatus(t, scanID, scanCompletedCh, scansRESTApi.ScanCompleted, fullScanWaitTime, 5)
	scanCompleted := <-scanCompletedCh
	assert.Assert(t, scanCompleted, "Full scan should be completed")
	log.Println("Waiting for full scan results...")
	time.Sleep(time.Duration(waitForResults) * time.Second)
	// Validate the results for full scan
	scanResults := getResultsNumberForScan(t, scanID)
	log.Println("Full scan results number is", scanResults)
	assert.Assert(t, scanResults == scanResultsNum, "Wrong number of scan results")
	incScanID := createIncScan(t)
	log.Printf("Waiting %d seconds for the incremental scan to complete...\n", incScanWaitTime)
	// Wait for the inc scan to finish. See it's completed successfully
	incScanCompletedCh := make(chan bool, 1)
	pollScanUntilStatus(t, incScanID, incScanCompletedCh, scansRESTApi.ScanCompleted, incScanWaitTime, 5)
	incScanCompleted := <-incScanCompletedCh
	assert.Assert(t, incScanCompleted, "Incremental scan should be completed")
	log.Println("Waiting for incremental scan results...")
	// Validate the results for inc scan
	time.Sleep(time.Duration(waitForResults) * time.Second)
	incScanResults := getResultsNumberForScan(t, incScanID)
	log.Println("Incremental scan results number is", incScanResults)
	assert.Assert(t, incScanResults == incScanResultsNum, "Wrong number of inc scan results")

	listScans(t)
	getScansTags(t)
}

func createScanSourcesFile(t *testing.T) string {
	// Create a full scan
	b := bytes.NewBufferString("")
	createCommand := createASTIntegrationTestCommand(t)
	createCommand.SetOut(b)
	err := execute(createCommand, "-v", "--format", "json", "scan", "create", "--input-file", "scan_payload.json", "--sources", "sources.zip")
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

func listScans(t *testing.T) {
	b := bytes.NewBufferString("")
	getAllCommand := createASTIntegrationTestCommand(t)
	getAllCommand.SetOut(b)
	var limit uint64 = 40
	var offset uint64 = 0

	lim := fmt.Sprintf("limit=%s", strconv.FormatUint(limit, 10))
	off := fmt.Sprintf("offset=%s", strconv.FormatUint(offset, 10))

	err := execute(getAllCommand, "-v", "--format", "json", "scan", "list", "--filter", lim, "--filter", off)
	assert.NilError(t, err, "Getting all scans should pass")
	// Read response from buffer
	var getAllJSON []byte
	getAllJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading all scans response JSON should pass")
	allScans := scansRESTApi.ScansCollectionResponseModel{}
	err = json.Unmarshal(getAllJSON, &allScans)
	assert.NilError(t, err, "Parsing all scans response JSON should pass")
}

func listScansList(t *testing.T) {
	getAllCommand := createASTIntegrationTestCommand(t)
	var limit uint64 = 40
	var offset uint64 = 0
	lim := fmt.Sprintf("limit=%s", strconv.FormatUint(limit, 10))
	off := fmt.Sprintf("offset=%s", strconv.FormatUint(offset, 10))
	err := execute(getAllCommand, "-v", "--format", "list", "scan", "list", "--filter", lim, "--filter", off)
	assert.NilError(t, err, "Getting all scans should pass")
}

func getScanByID(t *testing.T, scanID string) *scansRESTApi.ScanResponseModel {
	getBuffer := bytes.NewBufferString("")
	getCommand := createASTIntegrationTestCommand(t)
	getCommand.SetOut(getBuffer)
	err := execute(getCommand, "-v", "--format", "json", "scan", "show", scanID)
	assert.NilError(t, err)
	// Read response from buffer
	var getScanJSON []byte
	getScanJSON, err = ioutil.ReadAll(getBuffer)
	assert.NilError(t, err, "Reading scan response JSON should pass")
	getScan := scansRESTApi.ScanResponseModel{}
	err = json.Unmarshal(getScanJSON, &getScan)
	assert.NilError(t, err, "Parsing scan response JSON should pass")
	assert.Assert(t, cmp.Equal(getScan.ID, scanID))
	return &getScan
}
func getScanByIDList(t *testing.T, scanID string) {
	getCommand := createASTIntegrationTestCommand(t)
	err := execute(getCommand, "-v", "--format", "list", "scan", "show", scanID)
	assert.NilError(t, err)
}

func getScansTags(t *testing.T) {
	b := bytes.NewBufferString("")
	tagsCommand := createASTIntegrationTestCommand(t)
	tagsCommand.SetOut(b)
	err := execute(tagsCommand, "-v", "--format", "json", "scan", "tags")
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
	createIncCommand := createASTIntegrationTestCommand(t)
	createIncCommand.SetOut(incBuff)
	err := execute(createIncCommand, "-v", "--format", "json", "scan", "create", "--input-file", "scan_inc_payload.json", "--sources", "sources_inc.zip")
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

func pollScanUntilStatus(t *testing.T, scanID string, ch chan<- bool, requiredStatus scansRESTApi.ScanStatus, timeout, sleep int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	for {
		log.Printf("Polling scan %s\n", scanID)
		scan := getScanByID(t, scanID)
		getScanByIDList(t, scanID)
		if string(scan.Status) == string(requiredStatus) {
			ch <- true
			return
		} else if string(scan.Status) == scansRESTApi.ScanFailed || string(scan.Status) == scansRESTApi.ScanCanceled {
			ch <- false
			return
		} else {
			time.Sleep(time.Duration(sleep) * time.Second)
		}
	}

	<-ctx.Done()
	ch <- false
}
