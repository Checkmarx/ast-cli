//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"gotest.tools/assert"
)

// Read and unmarshall json from 'src' into 'dest'
func unmarshall(t *testing.T, src *bytes.Buffer, dest interface{}, msg string) []byte {
	//mu.Lock()
	//defer mu.Unlock()
	var responseJson []byte

	responseJson, err := ioutil.ReadAll(src)
	assert.NilError(t, err, msg)

	if len(responseJson) > 0 {
		err = json.Unmarshal(responseJson, dest)
		assert.NilError(t, err, msg)
	}

	return responseJson
}
