package util

import (
	"os"
	"strings"
	"testing"

	"gotest.tools/assert"
)

const mockFormatErrorMessage = "Invalid format MOCK"

func TestNewUtilsCommand(t *testing.T) {
	cmd := NewUtilsCommand(nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Utils command must exist")
}

func TestCompressFile_Success(t *testing.T) {
	_, err := CompressFile("package.json", "package.json", "cx-")
	assert.NilError(t, err, "CompressFile must run well")
}

func TestCompressFile_Fail(t *testing.T) {
	_, err := CompressFile("package.json", "package.json", "cx-")
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

func TestDeferCloseFileAndWriter_OnlyFile(t *testing.T) {
	file, err := os.OpenFile("../data/package.json", os.O_RDWR, 0644)
	assert.NilError(t, err, "OpenFile must run well")
	CloseFilesAndWriter(nil, file)
}

func TestCompressFile_EmptyDirectoryPrefix(t *testing.T) {
	outputFileName, err := CompressFile("testfile.txt", "output.zip", "")
	assert.NilError(t, err)
	// Assert that the output file name contains the default prefix
	assert.Assert(t, strings.Contains(outputFileName, "cx-"))
}
