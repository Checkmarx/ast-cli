package util

import (
	"archive/zip"
	"os"
	"path/filepath"
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
		&mock.JWTMockWrapper{},
		mock.ChatMockWrapper{},
		nil,
		nil,
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
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
	// Error message is platform-specific, just check that error exists
	assert.Assert(t, err != nil, "Expected error when reading non-existent file")
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

// TestIsDirOrSymLinkToDir_RegularDirectory tests with a regular directory
func TestIsDirOrSymLinkToDir_RegularDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-dir-*")
	assert.NilError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	fileInfo, err := os.Stat(tempDir)
	assert.NilError(t, err)

	isDir := IsDirOrSymLinkToDir(tempDir, fileInfo)
	assert.Assert(t, isDir, "Regular directory should return true")
}

// TestIsDirOrSymLinkToDir_RegularFile tests with a regular file
func TestIsDirOrSymLinkToDir_RegularFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-file-*.txt")
	assert.NilError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_ = tempFile.Close()

	fileInfo, err := os.Stat(tempFile.Name())
	assert.NilError(t, err)

	isDir := IsDirOrSymLinkToDir(tempFile.Name(), fileInfo)
	assert.Assert(t, !isDir, "Regular file should return false")
}

// TestIsDirOrSymLinkToDir_NestedDirectory tests with nested directory paths
func TestIsDirOrSymLinkToDir_NestedDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-nested-*")
	assert.NilError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	nestedDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(nestedDir, os.ModePerm)
	assert.NilError(t, err)

	fileInfo, err := os.Stat(nestedDir)
	assert.NilError(t, err)

	isDir := IsDirOrSymLinkToDir(tempDir, fileInfo)
	assert.Assert(t, isDir, "Nested directory should return true")
}

// TestIsDirOrSymLinkToDir_SymLinkToDirectory tests with symlink to directory
func TestIsDirOrSymLinkToDir_SymLinkToDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-target-*")
	assert.NilError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create symlink to directory
	linkPath := tempDir + "_link"
	err = os.Symlink(tempDir, linkPath)
	if err != nil {
		// Symlinks might not be available on all systems
		t.Skip("Symlinks not available on this system")
	}
	defer func() { _ = os.Remove(linkPath) }()

	fileInfo, err := os.Lstat(linkPath)
	assert.NilError(t, err)

	isDir := IsDirOrSymLinkToDir(tempDir+"_link", fileInfo)
	assert.Assert(t, isDir, "Symlink to directory should return true")
}

// TestIsDirOrSymLinkToDir_SymLinkToFile tests with symlink to file
func TestIsDirOrSymLinkToDir_SymLinkToFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-link-target-*.txt")
	assert.NilError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_ = tempFile.Close()

	// Create symlink to file
	linkPath := tempFile.Name() + "_link"
	err = os.Symlink(tempFile.Name(), linkPath)
	if err != nil {
		// Symlinks might not be available on all systems
		t.Skip("Symlinks not available on this system")
	}
	defer func() { _ = os.Remove(linkPath) }()

	fileInfo, err := os.Lstat(linkPath)
	assert.NilError(t, err)

	isDir := IsDirOrSymLinkToDir(tempFile.Name()+"_link", fileInfo)
	assert.Assert(t, !isDir, "Symlink to file should return false")
}

// TestIsGitURL_Extended tests more Git URL variations
func TestIsGitURL_Extended(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"HTTPS with .git", "https://github.com/user/repo.git", true},
		{"HTTPS without .git", "https://github.com/user/repo", true},
		{"SSH format", "git@github.com:user/repo.git", true},
		{"HTTP format", "http://example.com/repo.git", true},
		{"HTTPS with just host/path", "https://github.com/user", true},
		{"Invalid - no scheme", "github.com/user/repo", false},
		{"Invalid - random string", "not-a-url", false},
		{"SSH with host only", "git@github.com:repo", true},
		{"HTTP with host only", "http://example.com", true},
		{"Git prefix format", ":git:github.com/repo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGitURL(tt.url)
			assert.Equal(t, got, tt.expected, "URL: %s", tt.url)
		})
	}
}

// TestIsSSHURL_Extended tests more SSH URL variations
func TestIsSSHURL_Extended(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Standard SSH", "user@host:path/to/repo.git", true},
		{"SSH with port", "user@host:22/path/to/repo.git", true},
		{"SSH GitHub", "git@github.com:user/repo.git", true},
		{"SSH GitLab", "git@gitlab.com:user/repo.git", true},
		{"Invalid - no @", "user_host:path/to/repo", false},
		{"Invalid - no colon", "user@hostpath/to/repo", false},
		{"Invalid - no path", "user@host:", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSSHURL(tt.url)
			assert.Equal(t, got, tt.expected, "URL: %s", tt.url)
		})
	}
}

// TestCompressFile_WithValidFile tests CompressFile with a valid source file
func TestCompressFile_WithValidFile(t *testing.T) {
	// Create a temporary source file
	sourceFile, err := os.CreateTemp("", "source-*.txt")
	assert.NilError(t, err)
	defer func() { _ = os.Remove(sourceFile.Name()) }()

	_, err = sourceFile.WriteString("test content for compression")
	assert.NilError(t, err)
	_ = sourceFile.Close()

	// Compress the file
	zipPath, err := CompressFile(sourceFile.Name(), "compressed.txt", "test-")
	assert.NilError(t, err)
	defer func() { _ = os.Remove(zipPath) }()

	// Verify zip file exists and has content
	assert.Assert(t, zipPath != "")
	fileInfo, err := os.Stat(zipPath)
	assert.NilError(t, err)
	assert.Assert(t, fileInfo.Size() > 0, "Zip file should have content")
}

// TestCompressFile_WithCustomPrefix tests CompressFile with custom directory prefix
func TestCompressFile_WithCustomPrefix(t *testing.T) {
	sourceFile, err := os.CreateTemp("", "source-*.txt")
	assert.NilError(t, err)
	defer func() { _ = os.Remove(sourceFile.Name()) }()

	_, _ = sourceFile.WriteString("custom prefix test")
	_ = sourceFile.Close()

	zipPath, err := CompressFile(sourceFile.Name(), "output.txt", "myprefix-")
	assert.NilError(t, err)
	defer func() { _ = os.Remove(zipPath) }()

	assert.Assert(t, strings.Contains(zipPath, "myprefix-"), "Zip path should contain custom prefix")
}

// TestReadFileAsString_WithContent tests reading actual file content
func TestReadFileAsString_WithContent(t *testing.T) {
	// Create a test file with known content
	tempFile, err := os.CreateTemp("", "content-test-*.txt")
	assert.NilError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()

	content := "This is test content for reading"
	_, err = tempFile.WriteString(content)
	assert.NilError(t, err)
	_ = tempFile.Close()

	// Read the file
	readContent, err := ReadFileAsString(tempFile.Name())
	assert.NilError(t, err)
	assert.Equal(t, readContent, content)
}

// TestCloseOutputFile_WithValidFile tests CloseOutputFile with valid file
func TestCloseOutputFile_WithValidFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "valid-output-*.txt")
	assert.NilError(t, err)
	defer os.Remove(tempFile.Name())

	tempFile.WriteString("test data")

	// This should not panic
	CloseOutputFile(tempFile)
}

// TestCloseZipWriter_WithValidWriter tests CloseZipWriter with valid writer
func TestCloseZipWriter_WithValidWriter(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-zipwriter-*.zip")
	assert.NilError(t, err)
	defer os.Remove(tempFile.Name())

	zipWriter := zip.NewWriter(tempFile)

	// This should not panic
	CloseZipWriter(zipWriter, tempFile)
}

// TestExtractFolderNameFromZipPath_EdgeCases tests edge cases
func TestExtractFolderNameFromZipPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		outputFileName string
		dirPrefix      string
		shouldError    bool
	}{
		{"Empty filename", "", "cx-", true},
		{"Multiple occurrences of prefix", "cx-cx-archive.zip", "cx-", false},
		{"Prefix at end", "archive.zip", ".zip", false},
		{"No match found", "archive.zip", "cx-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := extractFolderNameFromZipPath(tt.outputFileName, tt.dirPrefix)
			if tt.shouldError {
				assert.Assert(t, err != nil, "Expected error for: %s", tt.name)
			}
		})
	}
}
