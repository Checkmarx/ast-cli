package util

import (
	"testing"

	"gotest.tools/assert"
)

const mockFormatErrorMessage = "Invalid format MOCK"

func TestNewUtilsCommand(t *testing.T) {
	cmd := NewUtilsCommand(nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Utils command must exist")
}

func TestCompressFile(t *testing.T) {
	_, err := CompressFile("package.json", "package.json")
	assert.NilError(t, err, "CompressFile must run well")
}

// test ReadFileAsString
func TestReadFileAsString_Success(t *testing.T) {
	_, err := ReadFileAsString("../data/package.json")
	assert.NilError(t, err, "ReadFileAsString must run well")
}

func TestReadFileAsString_NoFile_Fail(t *testing.T) {
	_, err := ReadFileAsString("no-file-exists-with-this-name.json")
	assert.Error(t, err, "open no-file-exists-with-this-name.json: no such file or directory")
}
