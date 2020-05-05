// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	rm "github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"

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
	e := Engines()
	assert.Assert(t, cmp.Equal(e.Waiting, 1))
	assert.Assert(t, cmp.Equal(e.Running, 0))
	s := Scans()
	assert.Assert(t, cmp.Equal(s.Waiting, 0))
	assert.Assert(t, cmp.Equal(s.Running, 0))

	scanID := createScanSourcesFile(t)
	waitTimeSec := 10
	scanCompletedCh := make(chan bool, 1)
	pollScanUntilStatus(t, scanID, scanCompletedCh, scansRESTApi.ScanPending, waitTimeSec, 5)
	assert.Assert(t, <-scanCompletedCh, "Scan should be queued")

	pollScanUntilStatus(t, scanID, scanCompletedCh, scansRESTApi.ScanRunning, waitTimeSec, 5)
	assert.Assert(t, <-scanCompletedCh, "Scan should be running")

	e = Engines()
	assert.Assert(t, cmp.Equal(e.Waiting, 0))
	assert.Assert(t, cmp.Equal(e.Running, 1))
	s = Scans()
	assert.Assert(t, cmp.Equal(s.Waiting, 0))
	assert.Assert(t, cmp.Equal(s.Running, 1))
}

func Scans() scans {
	scanCollection := rm.ScansCollection{}
	invokeCommand(t, &scanCollection, "sr", "scans")
	result := scans{}
	for i, s := range scanCollection.Scans {
		if s.State == rm.AllocatedScanState || s.State == rm.RunningScanState {
			result.Running++
		} else if s.State == rm.QueuedScanState {
			result.Waiting++
		}
	}
	return result
}

func Engines() engines {
	enginesCollection := rm.EnginesCollection{}
	invokeCommand(t, &enginesCollection, "sr", "engines")
	result := engines{}
	for i, engine := range enginesCollection.Engines {
		if engine.Status == rm.AllocatedEngineStatus || engine.Status == rm.BusyEngineStatus {
			result.Running++
		} else if engine.State == rm.ReadyEngineStatus {
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
	assert.NilError(t, err, "Reading scan response JSON should pass")
	err = json.Unmarshal(getScanJSON, result)
	assert.NilError(t, err, "Parsing scan response JSON should pass")
}
