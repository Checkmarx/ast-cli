package iacrealtime

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

func TestNewScanner(t *testing.T) {
	dm := NewDockerManager()
	scanner := NewScanner(dm)

	if scanner == nil {
		t.Error("NewScanner() should not return nil")
	}

	if scanner.dockerManager != dm {
		t.Error("NewScanner() should set dockerManager field")
	}
}

func TestScanner_extractErrorCode(t *testing.T) {
	scanner := NewScanner(NewDockerManager())

	tests := []struct {
		name     string
		msg      string
		expected string
	}{
		{
			name:     "Message with error code at end",
			msg:      "Docker command failed with exit code 60",
			expected: "60",
		},
		{
			name:     "Message with error code at end (different code)",
			msg:      "Command failed 50",
			expected: "50",
		},
		{
			name:     "Message without spaces",
			msg:      "error",
			expected: "",
		},
		{
			name:     "Empty message",
			msg:      "",
			expected: "",
		},
		{
			name:     "Message with multiple spaces",
			msg:      "Multiple words in error message 40",
			expected: "40",
		},
		{
			name:     "Message ending with non-numeric",
			msg:      "Error message end",
			expected: "end",
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := scanner.extractErrorCode(ttt.msg)
			if got != ttt.expected {
				t.Errorf("extractErrorCode() = %v, want %v", got, ttt.expected)
			}
		})
	}
}

func TestScanner_readKicsResultsFile(t *testing.T) {
	scanner := NewScanner(NewDockerManager())

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-scanner-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to cleanup temp dir: %v", err)
		}
	}()

	tests := []struct {
		name         string
		setupFile    func(string) error
		expectErr    bool
		expectedData *wrappers.KicsResultsCollection
	}{
		{
			name: "Valid KICS results file",
			setupFile: func(dir string) error {
				results := wrappers.KicsResultsCollection{
					Version: "1.0.0",
					Count:   1,
					Results: []wrappers.KicsQueries{
						{
							QueryName: "Test Query",
							Severity:  "high",
							Locations: []wrappers.KicsFiles{
								{
									SimilarityID: "test-id",
									Line:         10,
								},
							},
						},
					},
				}
				data, _ := json.Marshal(results)
				return os.WriteFile(filepath.Join(dir, ContainerResultsFileName), data, 0644)
			},
			expectErr: false,
			expectedData: &wrappers.KicsResultsCollection{
				Version: "1.0.0",
				Count:   1,
				Results: []wrappers.KicsQueries{
					{
						QueryName: "Test Query",
						Severity:  "high",
						Locations: []wrappers.KicsFiles{
							{
								SimilarityID: "test-id",
								Line:         10,
							},
						},
					},
				},
			},
		},
		{
			name: "Empty KICS results file",
			setupFile: func(dir string) error {
				results := wrappers.KicsResultsCollection{}
				data, _ := json.Marshal(results)
				return os.WriteFile(filepath.Join(dir, ContainerResultsFileName), data, 0644)
			},
			expectErr:    false,
			expectedData: &wrappers.KicsResultsCollection{},
		},
		{
			name: "Invalid JSON file",
			setupFile: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ContainerResultsFileName), []byte("invalid json"), 0644)
			},
			expectErr:    true,
			expectedData: nil,
		},
		{
			name: "Non-existent file",
			setupFile: func(dir string) error {
				// Don't create the file
				return nil
			},
			expectErr:    true,
			expectedData: nil,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Setup test directory
			testDir, err := os.MkdirTemp("", "test-read-*")
			if err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}
			defer func() {
				if err := os.RemoveAll(testDir); err != nil {
					t.Logf("Failed to cleanup test dir: %v", err)
				}
			}()

			// Setup the test file
			if err := ttt.setupFile(testDir); err != nil {
				t.Fatalf("Failed to setup test file: %v", err)
			}

			// Test the method
			result, err := scanner.readKicsResultsFile(testDir)

			if ttt.expectErr && err == nil {
				t.Error("readKicsResultsFile() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("readKicsResultsFile() unexpected error: %v", err)
			}

			if !ttt.expectErr && ttt.expectedData != nil {
				if result.Version != ttt.expectedData.Version {
					t.Errorf("Version = %v, want %v", result.Version, ttt.expectedData.Version)
				}
				if result.Count != ttt.expectedData.Count {
					t.Errorf("Count = %v, want %v", result.Count, ttt.expectedData.Count)
				}
				if len(result.Results) != len(ttt.expectedData.Results) {
					t.Errorf("Results length = %v, want %v", len(result.Results), len(ttt.expectedData.Results))
				}
			}
		})
	}
}

func TestScanner_HandleScanResult(t *testing.T) {
	scanner := NewScanner(NewDockerManager())

	// Create a temp directory with valid KICS results
	tempDir, err := os.MkdirTemp("", "test-handle-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to cleanup temp dir: %v", err)
		}
	}()

	// Create a valid results file
	results := wrappers.KicsResultsCollection{
		Results: []wrappers.KicsQueries{
			{
				QueryName: "Test Query",
				Severity:  "high",
				Locations: []wrappers.KicsFiles{
					{SimilarityID: "test-id", Line: 5},
				},
			},
		},
	}
	data, _ := json.Marshal(results)
	if err := os.WriteFile(filepath.Join(tempDir, ContainerResultsFileName), data, 0644); err != nil {
		t.Fatalf("Failed to create results file: %v", err)
	}

	tests := []struct {
		name       string
		err        error
		tempDir    string
		filePath   string
		expectErr  bool
		errorCheck func(error) bool
	}{
		{
			name:      "No error - success case",
			err:       nil,
			tempDir:   tempDir,
			filePath:  "test.yaml",
			expectErr: false,
		},
		{
			name:      "Invalid temp directory",
			err:       nil,
			tempDir:   "/non/existent/dir",
			filePath:  "test.yaml",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Mock the error type for specific test cases
			var testErr error = ttt.err

			// Create specific error types for testing
			if tt.name == "Engine not running error" {
				exitErr := &exec.ExitError{}
				// We can't easily set the exit code without reflection or other hacks
				// So we'll simulate this scenario differently
				testErr = exitErr
			} else if tt.name == "Invalid engine error" {
				testErr = &exec.Error{Name: "docker", Err: nil}
			} else if tt.name == "Generic error" {
				testErr = &exec.Error{Name: "docker", Err: nil}
			}

			results, err := scanner.HandleScanResult(testErr, ttt.tempDir, ttt.filePath)

			if ttt.expectErr && err == nil {
				t.Error("HandleScanResult() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("HandleScanResult() unexpected error: %v", err)
			}

			if !ttt.expectErr && err == nil {
				if results == nil {
					t.Error("HandleScanResult() should return results on success")
				}
			}

			if ttt.errorCheck != nil && err != nil {
				if !ttt.errorCheck(err) {
					t.Errorf("HandleScanResult() error doesn't match expected pattern: %v", err)
				}
			}
		})
	}
}

func TestScanner_processResults(t *testing.T) {
	scanner := NewScanner(NewDockerManager())

	tests := []struct {
		name      string
		setupFile func(string) error
		filePath  string
		expectErr bool
	}{
		{
			name: "Valid results processing",
			setupFile: func(dir string) error {
				results := wrappers.KicsResultsCollection{
					Results: []wrappers.KicsQueries{
						{
							QueryName:   "Test Query",
							Description: "Test Description",
							Severity:    "high",
							Locations: []wrappers.KicsFiles{
								{SimilarityID: "test-id", Line: 10},
							},
						},
					},
				}
				data, _ := json.Marshal(results)
				return os.WriteFile(filepath.Join(dir, ContainerResultsFileName), data, 0644)
			},
			filePath:  "test.yaml",
			expectErr: false,
		},
		{
			name: "Empty results processing",
			setupFile: func(dir string) error {
				results := wrappers.KicsResultsCollection{}
				data, _ := json.Marshal(results)
				return os.WriteFile(filepath.Join(dir, ContainerResultsFileName), data, 0644)
			},
			filePath:  "test.yaml",
			expectErr: false,
		},
		{
			name: "Invalid results file",
			setupFile: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ContainerResultsFileName), []byte("invalid"), 0644)
			},
			filePath:  "test.yaml",
			expectErr: true,
		},
		{
			name: "Missing results file",
			setupFile: func(dir string) error {
				// Don't create file
				return nil
			},
			filePath:  "test.yaml",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir, err := os.MkdirTemp("", "test-process-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					t.Logf("Failed to cleanup temp dir: %v", err)
				}
			}()

			// Setup test file
			if err := ttt.setupFile(tempDir); err != nil {
				t.Fatalf("Failed to setup test file: %v", err)
			}

			// Test the method
			results, err := scanner.processResults(tempDir, ttt.filePath)

			if ttt.expectErr && err == nil {
				t.Error("processResults() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("processResults() unexpected error: %v", err)
			}

			if !ttt.expectErr {
				if results == nil {
					t.Error("processResults() should return results slice")
				}
			}
		})
	}
}

func TestScanner_RunScan(t *testing.T) {
	// This test mainly verifies the integration between RunScan and HandleScanResult
	// The actual Docker execution will likely fail in test environment
	scanner := NewScanner(NewDockerManager())

	// Create temp directory with results file for successful processing
	tempDir, err := os.MkdirTemp("", "test-run-scan-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to cleanup temp dir: %v", err)
		}
	}()

	// Create a valid results file
	results := wrappers.KicsResultsCollection{
		Results: []wrappers.KicsQueries{
			{
				QueryName: "Test Query",
				Severity:  "medium",
				Locations: []wrappers.KicsFiles{
					{SimilarityID: "scan-test-id", Line: 15},
				},
			},
		},
	}
	data, _ := json.Marshal(results)
	if err := os.WriteFile(filepath.Join(tempDir, ContainerResultsFileName), data, 0644); err != nil {
		t.Fatalf("Failed to create results file: %v", err)
	}

	tests := []struct {
		name      string
		engine    string
		volumeMap string
		tempDir   string
		filePath  string
		expectErr bool
	}{
		{
			name:      "Invalid engine",
			engine:    "invalid-engine",
			volumeMap: "/tmp:/path",
			tempDir:   tempDir,
			filePath:  "test.yaml",
			expectErr: true,
		},
		{
			name:      "Empty engine",
			engine:    "",
			volumeMap: "/tmp:/path",
			tempDir:   tempDir,
			filePath:  "test.yaml",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := scanner.RunScan(ttt.engine, ttt.volumeMap, ttt.tempDir, ttt.filePath)

			if ttt.expectErr && err == nil {
				t.Error("RunScan() expected error but got none (docker might be available)")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("RunScan() unexpected error: %v", err)
			}
		})
	}
}

func TestScanner_Integration(t *testing.T) {
	// Test the full scanner workflow
	scanner := NewScanner(NewDockerManager())

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-scanner-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to cleanup temp dir: %v", err)
		}
	}()

	// Create test results file
	results := wrappers.KicsResultsCollection{
		Version: "1.6.0",
		Count:   2,
		Results: []wrappers.KicsQueries{
			{
				QueryName:   "Insecure Configuration",
				Description: "Configuration has security issues",
				Severity:    "critical",
				Platform:    "Kubernetes",
				Category:    "Security",
				Locations: []wrappers.KicsFiles{
					{
						SimilarityID: "integration-test-1",
						Line:         25,
					},
				},
			},
			{
				QueryName:   "Performance Issue",
				Description: "Configuration may impact performance",
				Severity:    "low",
				Platform:    "Docker",
				Category:    "Performance",
				Locations: []wrappers.KicsFiles{
					{
						SimilarityID: "integration-test-2",
						Line:         30,
					},
					{
						SimilarityID: "integration-test-3",
						Line:         35,
					},
				},
			},
		},
	}

	data, _ := json.Marshal(results)
	if err := os.WriteFile(filepath.Join(tempDir, ContainerResultsFileName), data, 0644); err != nil {
		t.Fatalf("Failed to create results file: %v", err)
	}

	// Test processResults directly since RunScan would try to execute Docker
	iacResults, err := scanner.processResults(tempDir, "integration-test.yaml")
	if err != nil {
		t.Fatalf("processResults() failed: %v", err)
	}

	// Verify results were processed correctly
	if len(iacResults) != 3 { // 2 queries with 1 and 2 locations respectively
		t.Errorf("Expected 3 IAC results, got %d", len(iacResults))
	}

	// Verify first result
	if len(iacResults) > 0 {
		result := iacResults[0]
		if result.Title != "Insecure Configuration" {
			t.Errorf("Expected title 'Insecure Configuration', got '%s'", result.Title)
		}
		if result.Severity != "Critical" {
			t.Errorf("Expected severity 'Critical', got '%s'", result.Severity)
		}
		if result.SimilarityID != "integration-test-1" {
			t.Errorf("Expected similarity ID 'integration-test-1', got '%s'", result.SimilarityID)
		}
	}

	// Test error code extraction
	testCodes := map[string]string{
		"Command failed with exit 60": "60",
		"Docker error 50":             "50",
		"No error code":               "code",
		"":                            "",
	}

	for msg, expected := range testCodes {
		got := scanner.extractErrorCode(msg)
		if got != expected {
			t.Errorf("extractErrorCode(%q) = %q, want %q", msg, got, expected)
		}
	}
}
