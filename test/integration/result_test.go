// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	resultsRaw "github.com/checkmarxDev/sast-results/pkg/web/path/raw"
	"io/ioutil"
	"strconv"
	"testing"

	"gotest.tools/assert"
)

func getResultsNumberForScan(t *testing.T, scanID string) uint {
	b := bytes.NewBufferString("")
	getResultsCmd := createASTIntegrationTestCommand(t)
	getResultsCmd.SetOut(b)
	var limit uint64 = 600
	var offset uint64 = 0
	lim := fmt.Sprintf("limit=%s", strconv.FormatUint(limit, 10))
	off := fmt.Sprintf("offset=%s", strconv.FormatUint(offset, 10))

	err := execute(getResultsCmd, "-v", "--format", "json", "result", "list", scanID, "--filter", lim, "--filter", off)
	assert.NilError(t, err, "Getting all results should pass")
	// Read response from buffer
	var getAllJSON []byte
	getAllJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading all results response JSON should pass")
	resultsReponseModel := resultsRaw.ResultsCollection{}
	err = json.Unmarshal(getAllJSON, &resultsReponseModel)
	assert.NilError(t, err, "Parsing all results response JSON should pass")
	return resultsReponseModel.TotalCount
}
