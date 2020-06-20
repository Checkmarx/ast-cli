// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	rm "github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"
	"github.com/spf13/viper"
	"io/ioutil"
	"testing"
	"time"

	"gotest.tools/assert/cmp"

	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/v1/rest"
	"gotest.tools/assert"
)

type engines struct {
	Waiting int
	Running int
}

type scans struct {
	Waiting int
	Running int
}

func TestSastResourceE2E(t *testing.T) {
	e := Engines(t)
	assert.Assert(t, cmp.Equal(e.Waiting, 3))
	assert.Assert(t, cmp.Equal(e.Running, 0))
	s := Scans(t)
	assert.Assert(t, cmp.Equal(s.Waiting, 0))
	assert.Assert(t, cmp.Equal(s.Running, 0))

	scanID, projectID := createScanSourcesFile(t)
	defer deleteProject(t, projectID)
	defer deleteScan(t, scanID)

	waitTimeSec := viper.GetInt("TEST_FULL_SCAN_WAIT_COMPLETED_SECONDS")
	scanStatusAsWanted := pollScanUntilStatus(t, scanID, scansRESTApi.ScanRunning, waitTimeSec, 5)
	assert.Assert(t, scanStatusAsWanted, "Scan should be running")

	// Let the sr to update
	time.Sleep(5 * time.Second)
	e = Engines(t)
	assert.Assert(t, cmp.Equal(e.Waiting, 2))
	assert.Assert(t, cmp.Equal(e.Running, 1))
	s = Scans(t)
	assert.Assert(t, cmp.Equal(s.Waiting, 0))
	assert.Assert(t, cmp.Equal(s.Running, 1))
}

func Scans(t *testing.T) scans {
	var scanCollection []rm.Scan
	invokeCommand(t, &scanCollection, "--format", "json", "sr", "scans")
	result := scans{}
	for _, s := range scanCollection {
		if s.State == rm.AllocatedScanState || s.State == rm.RunningScanState {
			result.Running++
		} else if s.State == rm.QueuedScanState {
			result.Waiting++
		}
	}
	return result
}

func Engines(t *testing.T) engines {
	var enginesCollection []rm.Engine
	invokeCommand(t, &enginesCollection, "--format", "json", "sr", "engines")
	result := engines{}
	for _, engine := range enginesCollection {
		if engine.Status == rm.AllocatedEngineStatus || engine.Status == rm.BusyEngineStatus {
			result.Running++
		} else if engine.Status == rm.ReadyEngineStatus {
			result.Waiting++
		}
	}
	return result
}

func invokeCommand(t *testing.T, result interface{}, params ...string) {
	getBuffer := bytes.NewBufferString("")
	getCommand := createASTIntegrationTestCommand(t)
	getCommand.SetOut(getBuffer)
	err := execute(getCommand, params...)
	assert.NilError(t, err)
	// Read response from buffer
	var getScanJSON []byte
	getScanJSON, err = ioutil.ReadAll(getBuffer)
	fmt.Println("JSON:", string(getScanJSON))
	assert.NilError(t, err, "Reading scan response JSON should pass")
	err = json.Unmarshal(getScanJSON, result)
	assert.NilError(t, err, "Parsing scan response JSON should pass")
}
