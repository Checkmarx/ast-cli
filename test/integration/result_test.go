// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

func getResultsNumberForScan(t *testing.T, scanID string) int {
	b := bytes.NewBufferString("")
	getResultsCmd := createASTIntegrationTestCommand(t)
	getResultsCmd.SetOut(b)
	var limit uint64 = 600
	var offset uint64 = 0
	l := strconv.FormatUint(limit, 10)
	o := strconv.FormatUint(offset, 10)
	err := execute(getResultsCmd, "-v", "result", "list", scanID, "--limit", l, "--offset", o)
	assert.NilError(t, err, "Getting all results should pass")
	// Read response from buffer
	var getAllJSON []byte
	getAllJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading all results response JSON should pass")
	allResults := []wrappers.ResultResponseModel{}
	err = json.Unmarshal(getAllJSON, &allResults)
	assert.NilError(t, err, "Parsing all results response JSON should pass")
	return len(allResults)
}
