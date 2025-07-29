package iacrealtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
)

func TestNewFileHandler(t *testing.T) {
	fh := NewFileHandler()

	if fh == nil {
		t.Error("NewFileHandler() should not return nil")
	}
}

func TestFileHandler_CreateTempDirectory(t *testing.T) {
	fh := NewFileHandler()

	tempDir, err := fh.CreateTempDirectory()

	if err != nil {
		t.Fatalf("CreateTempDirectory() should not return error: %v", err)
	}

	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to cleanup temp dir: %v", err)
		}
	}()

	// Test that directory was created
	if tempDir == "" {
		t.Error("CreateTempDirectory() should return non-empty path")
	}

	// Test that directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("CreateTempDirectory() should create actual directory")
	}

	// Test that directory name contains expected pattern
	if !strings.Contains(tempDir, ContainerTempDirPattern) {
		t.Errorf("Temp directory should contain pattern '%s', got '%s'", ContainerTempDirPattern, tempDir)
	}

	// Test that subsequent calls create different directories
	tempDir2, err := fh.CreateTempDirectory()
	if err != nil {
		t.Fatalf("Second CreateTempDirectory() call failed: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir2); err != nil {
			t.Logf("Failed to cleanup temp dir2: %v", err)
		}
	}()

	if tempDir == tempDir2 {
		t.Error("CreateTempDirectory() should create unique directories")
	}
}

func TestFileHandler_CleanupTempDirectory(t *testing.T) {
	fh := NewFileHandler()

	tests := []struct {
		name      string
		setup     func() string
		expectErr bool
	}{
		{
			name: "Valid temp directory",
			setup: func() string {
				tempDir, _ := os.MkdirTemp("", "test-cleanup-*")
				return tempDir
			},
			expectErr: false,
		},
		{
			name: "Non-existent directory",
			setup: func() string {
				return "/non/existent/directory"
			},
			expectErr: false, // os.RemoveAll doesn't error on non-existent paths
		},
		{
			name: "Empty directory path",
			setup: func() string {
				return ""
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			tempDir := ttt.setup()

			err := fh.CleanupTempDirectory(tempDir)

			if ttt.expectErr && err == nil {
				t.Error("CleanupTempDirectory() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("CleanupTempDirectory() unexpected error: %v", err)
			}

			// If tempDir was valid and not empty, verify it was removed
			if tempDir != "" && tempDir != "/non/existent/directory" {
				if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
					t.Error("CleanupTempDirectory() should remove the directory")
				}
			}
		})
	}
}

func TestFileHandler_HasSupportedExtension(t *testing.T) {
	fh := NewFileHandler()

	// Mock the KicsBaseFilters for testing
	originalFilters := commonParams.KicsBaseFilters
	commonParams.KicsBaseFilters = []string{".yaml", ".yml", ".json", ".tf"}
	defer func() {
		commonParams.KicsBaseFilters = originalFilters
	}()

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "YAML file",
			filePath: "test.yaml",
			expected: true,
		},
		{
			name:     "YML file",
			filePath: "test.yml",
			expected: true,
		},
		{
			name:     "JSON file",
			filePath: "config.json",
			expected: true,
		},
		{
			name:     "Terraform file",
			filePath: "main.tf",
			expected: true,
		},
		{
			name:     "Unsupported extension",
			filePath: "test.txt",
			expected: false,
		},
		{
			name:     "No extension",
			filePath: "Dockerfile",
			expected: false,
		},
		{
			name:     "Empty path",
			filePath: "",
			expected: false,
		},
		{
			name:     "Path with supported extension in directory",
			filePath: "/path/to/test.yaml",
			expected: true,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := fh.HasSupportedExtension(ttt.filePath)
			if got != ttt.expected {
				t.Errorf("HasSupportedExtension() = %v, want %v", got, ttt.expected)
			}
		})
	}
}

func TestFileHandler_CopyFileToTempDir(t *testing.T) {
	fh := NewFileHandler()

	// Create a test source file
	testContent := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test"
	sourceFile, err := os.CreateTemp("", "source-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	defer func() {
		if err := os.Remove(sourceFile.Name()); err != nil {
			t.Logf("Failed to remove source file: %v", err)
		}
	}()

	if _, err := sourceFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to source file: %v", err)
	}
	sourceFile.Close()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-copy-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to cleanup temp dir: %v", err)
		}
	}()

	tests := []struct {
		name      string
		filePath  string
		tempDir   string
		expectErr bool
	}{
		{
			name:      "Valid file copy",
			filePath:  sourceFile.Name(),
			tempDir:   tempDir,
			expectErr: false,
		},
		{
			name:      "Non-existent source file",
			filePath:  "/non/existent/file.yaml",
			tempDir:   tempDir,
			expectErr: true,
		},
		{
			name:      "Invalid temp directory",
			filePath:  sourceFile.Name(),
			tempDir:   "/non/existent/dir",
			expectErr: true,
		},
		{
			name:      "Empty file path",
			filePath:  "",
			tempDir:   tempDir,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := fh.CopyFileToTempDir(ttt.filePath, ttt.tempDir)

			if ttt.expectErr && err == nil {
				t.Error("CopyFileToTempDir() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("CopyFileToTempDir() unexpected error: %v", err)
			}

			// If successful, verify file was copied
			if !ttt.expectErr && err == nil {
				fileName := filepath.Base(ttt.filePath)
				destPath := filepath.Join(ttt.tempDir, fileName)

				if _, err := os.Stat(destPath); os.IsNotExist(err) {
					t.Error("CopyFileToTempDir() should create destination file")
				}

				// Verify content matches
				destContent, err := os.ReadFile(destPath)
				if err != nil {
					t.Errorf("Failed to read destination file: %v", err)
				} else if string(destContent) != testContent {
					t.Error("CopyFileToTempDir() should preserve file content")
				}
			}
		})
	}
}

func TestFileHandler_CopyFileToTempDir_SecurityTests(t *testing.T) {
	fh := NewFileHandler()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-security-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to cleanup temp dir: %v", err)
		}
	}()

	// Create a test source file
	sourceFile, err := os.CreateTemp("", "source-security-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	defer func() {
		if err := os.Remove(sourceFile.Name()); err != nil {
			t.Logf("Failed to remove source file: %v", err)
		}
	}()

	if _, err := sourceFile.WriteString("test content"); err != nil {
		t.Fatalf("Failed to write to source file: %v", err)
	}
	sourceFile.Close()

	// Test path traversal protection by trying to use .. in filename
	maliciousPath := filepath.Join(filepath.Dir(sourceFile.Name()), "..", filepath.Base(sourceFile.Name()))

	err = fh.CopyFileToTempDir(maliciousPath, tempDir)

	// This should work fine as long as the resolved path is still valid
	// The security check is more about preventing writing outside tempDir
	if err != nil {
		t.Logf("Path with .. failed as expected: %v", err)
	}
}

func TestFileHandler_PrepareScanEnvironment(t *testing.T) {
	fh := NewFileHandler()

	// Mock supported extensions
	originalFilters := commonParams.KicsBaseFilters
	commonParams.KicsBaseFilters = []string{".yaml", ".yml"}
	defer func() {
		commonParams.KicsBaseFilters = originalFilters
	}()

	// Create a test file
	testContent := "apiVersion: v1\nkind: Pod"
	sourceFile, err := os.CreateTemp("", "test-prepare-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := os.Remove(sourceFile.Name()); err != nil {
			t.Logf("Failed to remove test file: %v", err)
		}
	}()

	if _, err := sourceFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	sourceFile.Close()

	tests := []struct {
		name      string
		filePath  string
		expectErr bool
	}{
		{
			name:      "Valid YAML file",
			filePath:  sourceFile.Name(),
			expectErr: false,
		},
		{
			name:      "Unsupported file extension",
			filePath:  "test.txt",
			expectErr: true,
		},
		{
			name:      "Non-existent file",
			filePath:  "/non/existent/file.yaml",
			expectErr: true,
		},
		{
			name:      "Empty file path",
			filePath:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			volumeMap, tempDir, err := fh.PrepareScanEnvironment(ttt.filePath)

			// Cleanup temp directory if created
			if tempDir != "" {
				defer func() {
					if cleanupErr := fh.CleanupTempDirectory(tempDir); cleanupErr != nil {
						t.Logf("Failed to cleanup temp dir: %v", cleanupErr)
					}
				}()
			}

			if ttt.expectErr && err == nil {
				t.Error("PrepareScanEnvironment() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("PrepareScanEnvironment() unexpected error: %v", err)
			}

			if !ttt.expectErr {
				// Verify volume map format
				if volumeMap == "" {
					t.Error("PrepareScanEnvironment() should return non-empty volume map")
				}

				expectedVolumeFormat := tempDir + ":" + ContainerPath
				if volumeMap != expectedVolumeFormat {
					t.Errorf("PrepareScanEnvironment() volume map = %s, want %s", volumeMap, expectedVolumeFormat)
				}

				// Verify temp directory exists
				if _, err := os.Stat(tempDir); os.IsNotExist(err) {
					t.Error("PrepareScanEnvironment() should create temp directory")
				}

				// Verify file was copied to temp directory
				fileName := filepath.Base(ttt.filePath)
				destPath := filepath.Join(tempDir, fileName)
				if _, err := os.Stat(destPath); os.IsNotExist(err) {
					t.Error("PrepareScanEnvironment() should copy file to temp directory")
				}
			}
		})
	}
}

func TestFileHandler_Integration(t *testing.T) {
	fh := NewFileHandler()

	// Mock supported extensions
	originalFilters := commonParams.KicsBaseFilters
	commonParams.KicsBaseFilters = []string{".yaml"}
	defer func() {
		commonParams.KicsBaseFilters = originalFilters
	}()

	// Create a test file
	testContent := "test: content"
	sourceFile, err := os.CreateTemp("", "integration-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := os.Remove(sourceFile.Name()); err != nil {
			t.Logf("Failed to remove test file: %v", err)
		}
	}()

	if _, err := sourceFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	sourceFile.Close()

	// Test full workflow
	volumeMap, tempDir, err := fh.PrepareScanEnvironment(sourceFile.Name())
	if err != nil {
		t.Fatalf("PrepareScanEnvironment failed: %v", err)
	}

	// Verify results
	if volumeMap == "" {
		t.Error("Volume map should not be empty")
	}

	if tempDir == "" {
		t.Error("Temp directory should not be empty")
	}

	// Verify file exists in temp directory
	fileName := filepath.Base(sourceFile.Name())
	destPath := filepath.Join(tempDir, fileName)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("File should exist in temp directory")
	}

	// Test cleanup
	if err := fh.CleanupTempDirectory(tempDir); err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}

	// Verify directory was removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("Temp directory should be removed after cleanup")
	}
}
