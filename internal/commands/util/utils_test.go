package util

import (
	"archive/zip"
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

func TestCompressFile_EmptyDirectoryPrefix(t *testing.T) {
	outputFileName, err := CompressFile("testfile.txt", "output.zip", "")
	assert.NilError(t, err)
	// Assert that the output file name contains the default prefix
	assert.Assert(t, strings.Contains(outputFileName, "cx-"))
}

func TestCloseOutputFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-output-file-*.txt")
	assert.NilError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	CloseOutputFile(tempFile)
	closedErr := tempFile.Close()
	assert.ErrorContains(t, closedErr, "file already closed")
}

func TestCloseZipWriter(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test-zip-file-*.zip")
	assert.NilError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	zipWriter := zip.NewWriter(tempFile)

	CloseZipWriter(zipWriter, tempFile)
	zipClosedErr := zipWriter.Close()
	assert.ErrorContains(t, zipClosedErr, "zip: writer closed twice")
}

func TestCloseDataFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-data-file-*.txt")
	assert.NilError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	CloseDataFile(tempFile)
	closedErr := tempFile.Close()
	assert.ErrorContains(t, closedErr, "file already closed")
}
