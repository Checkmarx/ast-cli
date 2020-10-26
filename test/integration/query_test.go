// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"github.com/checkmarxDev/ast-cli/internal/commands"
	"gotest.tools/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestQueriesE2E(t *testing.T) {
	// List
	outBuff := bytes.NewBufferString("")
	cmd := createASTIntegrationTestCommand(t)
	cmd.SetOut(outBuff)
	err := execute(cmd, "-v", "--format", "json", "query", "list")
	listJSON, err := ioutil.ReadAll(outBuff)
	assert.NilError(t, err, "Reading list response JSON should pass")
	var listModel []*commands.QueryRepoView
	err = json.Unmarshal(listJSON, &listModel)
	assert.NilError(t, err, "Parsing list response JSON should pass")
	assert.Assert(t, len(listModel) > 0)
	var activeFound bool
	var activeRepoName string
	for _, r := range listModel {
		if r.IsActive == "active" {
			assert.Assert(t, !activeFound, "There should'nt be more than one active queries repo")
			activeFound = true
			activeRepoName = r.Name
		}
	}

	assert.Assert(t, activeFound, "Should be one active queries repo")

	// Download
	err = execute(cmd, "-v", "query", "download")
	assert.NilError(t, err)
	pwd, _ := os.Getwd()
	downloadedRepoPth := filepath.Join(pwd, commands.QueriesRepoDestFileName)
	defer os.Remove(downloadedRepoPth)
	stat, err := os.Stat(downloadedRepoPth)
	assert.NilError(t, err, "Stat downloaded repo tarball should pass")
	assert.Assert(t, stat.Size() > 1000) // > 1KB

	// Upload
	repoName := "cli_test"
	err = execute(cmd, "-v", "query", "upload", downloadedRepoPth, "--name", repoName)
	assert.NilError(t, err)
	outBuff.Reset()
	err = execute(cmd, "-v", "--format", "json", "query", "list")
	listJSON, err = ioutil.ReadAll(outBuff)
	assert.NilError(t, err, "Reading list response JSON should pass")
	listModel = nil
	err = json.Unmarshal(listJSON, &listModel)
	assert.NilError(t, err, "Parsing list response JSON should pass")
	var newRepoFound bool
	for _, r := range listModel {
		if r.Name == repoName {
			if r.IsActive == "active" {
				t.Error("Uploaded repo should not be active")
			}
			newRepoFound = true
			break
		}
	}

	assert.Assert(t, newRepoFound, "New repo should exists on the list response")

	// Activate
	err = execute(cmd, "-v", "query", "activate", repoName)
	assert.NilError(t, err)
	outBuff.Reset()
	err = execute(cmd, "-v", "--format", "json", "query", "list")
	listJSON, err = ioutil.ReadAll(outBuff)
	assert.NilError(t, err, "Reading list response JSON should pass")
	listModel = nil
	err = json.Unmarshal(listJSON, &listModel)
	assert.NilError(t, err, "Parsing list response JSON should pass")
	for _, r := range listModel {
		if r.Name == repoName {
			if r.IsActive == "inactive" {
				t.Error("Uploaded repo should be active")
			}
			break
		}
	}

	err = execute(cmd, "-v", "query", "activate", activeRepoName) // rollback
	assert.NilError(t, err)

	// Delete
	err = execute(cmd, "-v", "query", "delete", repoName)
	assert.NilError(t, err)
	outBuff.Reset()
	err = execute(cmd, "-v", "--format", "json", "query", "list")
	listJSON, err = ioutil.ReadAll(outBuff)
	assert.NilError(t, err, "Reading list response JSON should pass")
	listModel = nil
	err = json.Unmarshal(listJSON, &listModel)
	assert.NilError(t, err, "Parsing list response JSON should pass")
	for _, r := range listModel {
		if r.Name == repoName {
			t.Error("Uploaded repo should be deleted")
			break
		}
	}
}
