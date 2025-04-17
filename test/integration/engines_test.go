//go:build integration

package integration

import (
	"gotest.tools/assert"
	"testing"
)

func TestGetAllEnginesApiList(t *testing.T) {
	args := []string{
		"engines", "list-api",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_HelpSuccess(t *testing.T) {
	args := []string{
		"engines",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestGetSASTEnginesApiList_Success(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engine-name", "SAST",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestGetSCAEnginesApiList_Success(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engine-name", "SCA",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiListInvalidFlagDetails_Error(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engine-name", "xyz",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiListInTableFormat_Success(t *testing.T) {
	args := []string{
		"engines", "list-api", "--output-format", "table", "--engine-name", "SAST",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiListInJsonFormat_Success(t *testing.T) {
	args := []string{
		"engines", "list-api", "--output-format", "json", "--engine-name", "SAST",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}
