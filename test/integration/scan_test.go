package integration

import (
	"bytes"
	"io/ioutil"
	"testing"

	"gotest.tools/assert"
)

// Create a scan. Get it's Id. Keep polling and see it's completed. Then run incremental scan
// Get the tags from the scan

func TestRunScanSourcesFile(t *testing.T) {
	b := bytes.NewBufferString("")
	astRootCommand := createASTIntegrationTestCommand()
	astRootCommand.SetOut(b)
	_ = execute(astRootCommand, "-v", "scan", "create", "--inputFile", "scan_payload.json", "--sources", "sources.zip")
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
	scanID := string(out)
	println(scanID)
	assert.NilError(t, err)
}

func TestRunCreateScanCommandWithNoInput(t *testing.T) {
	astRootCommand := createASTIntegrationTestCommand()
	err := execute(astRootCommand, "-v", "scan", "create")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed creating a scan: no input was given\n")
}
