package util

import (
	"archive/zip"
	"os"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"gotest.tools/assert"
)

const mockFormatErrorMessage = "Invalid format MOCK"

func TestNewUtilsCommand(t *testing.T) {
	cmd := NewUtilsCommand(mock.GitHubMockWrapper{},
		mock.AzureMockWrapper{},
		mock.BitBucketMockWrapper{},
		nil,
		mock.GitLabMockWrapper{},
		nil,
		mock.LearnMoreMockWrapper{},
		mock.TenantConfigurationMockWrapper{},
		mock.ChatMockWrapper{},
		nil,
		nil,
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.ByorMockWrapper{},
		&mock.FeatureFlagsMockWrapper{})

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

func TestExtractFolderNameFromZipPath(t *testing.T) {
	type TestCase struct {
		Name           string
		OutputFileName string
		DirPrefix      string
		ExpectedResult string
		ExpectedError  string
	}
	testCases := []TestCase{
		{
			Name:           "Success: Standard Prefix",
			OutputFileName: "cx-archive.zip",
			DirPrefix:      "cx-",
			ExpectedResult: "cx-archive",
			ExpectedError:  "",
		},
		{
			Name:           "Success: Custom Prefix",
			OutputFileName: "my-archive.zip",
			DirPrefix:      "my-",
			ExpectedResult: "my-archive",
			ExpectedError:  "",
		},
		{
			Name:           "Failure: No Prefix Match",
			OutputFileName: "archive.zip",
			DirPrefix:      "cx-",
			ExpectedResult: "",
			ExpectedError:  "Failed to extract folder name from zip path: archive.zip with prefix: cx-",
		},
		{
			Name:           "Failure: Prefix Not Found",
			OutputFileName: "example.zip",
			DirPrefix:      "cx-",
			ExpectedResult: "",
			ExpectedError:  "Failed to extract folder name from zip path: example.zip with prefix: cx-",
		},
		{
			Name:           "Success: Full Name With Prefix",
			OutputFileName: "cx-archive.zip",
			DirPrefix:      "cx-",
			ExpectedResult: "cx-archive",
			ExpectedError:  "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			folderName, err := extractFolderNameFromZipPath(tc.OutputFileName, tc.DirPrefix)
			if tc.ExpectedError != "" {
				assert.ErrorContains(t, err, tc.ExpectedError)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, tc.ExpectedResult, folderName)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		array  []string
		val    string
		exists bool
	}{
		{"Value exists in array", []string{"a", "b", "c"}, "b", true},
		{"Value does not exist in array", []string{"x", "y", "z"}, "a", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Contains(tt.array, tt.val); got != tt.exists {
				t.Errorf("Contains() = %v, want %v", got, tt.exists)
			}
		})
	}
}

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Valid Git URL", "https://github.com/username/repository.git", true},
		{"Invalid Git URL", "notagiturl", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGitURL(tt.url); got != tt.expected {
				t.Errorf("IsGitURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsSSHURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Valid SSH URL", "user@host:path/to/repository.git", true},
		{"Invalid SSH URL", "notasshurl", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSSHURL(tt.url); got != tt.expected {
				t.Errorf("IsSSHURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}
