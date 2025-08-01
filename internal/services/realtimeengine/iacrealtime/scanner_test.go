package iacrealtime

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

func TestNewScanner(t *testing.T) {
	dm := NewMockContainerManager()
	scanner := NewScanner(dm)

	if scanner == nil {
		t.Error("NewScanner() should not return nil")
		t.FailNow()
	}

	if scanner.dockerManager == nil {
		t.Error("Scanner should have a container manager")
	}

	if scanner.mapper == nil {
		t.Error("Scanner should have a mapper")
	}
}

func TestScanner_extractErrorCode(t *testing.T) {
	scanner := NewScanner(NewMockContainerManager())

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

// Helper function to create valid KICS results data
func createValidKicsResults() wrappers.KicsResultsCollection {
	return wrappers.KicsResultsCollection{
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
}

// Helper function to create test file with valid JSON
func createValidKicsFile(dir string) error {
	results := createValidKicsResults()
	data, _ := json.Marshal(results)
	return os.WriteFile(filepath.Join(dir, ContainerResultsFileName), data, 0644)
}

// Helper function to create test file with empty results
func createEmptyKicsFile(dir string) error {
	results := wrappers.KicsResultsCollection{}
	data, _ := json.Marshal(results)
	return os.WriteFile(filepath.Join(dir, ContainerResultsFileName), data, 0644)
}

// Helper function to create test file with invalid JSON
func createInvalidKicsFile(dir string) error {
	return os.WriteFile(filepath.Join(dir, ContainerResultsFileName), []byte("invalid json"), 0644)
}

// Helper function to skip file creation (for non-existent file test)
func skipFileCreation(dir string) error {
	return nil
}

// Helper function to validate KICS results
func validateKicsResults(t *testing.T, result, expected *wrappers.KicsResultsCollection) {
	if expected == nil {
		return
	}

	if result.Version != expected.Version {
		t.Errorf("Version = %v, want %v", result.Version, expected.Version)
	}
	if result.Count != expected.Count {
		t.Errorf("Count = %v, want %v", result.Count, expected.Count)
	}
	if len(result.Results) != len(expected.Results) {
		t.Errorf("Results length = %v, want %v", len(result.Results), len(expected.Results))
	}
}

// Test case structure for readKicsResultsFile
type readKicsTestCase struct {
	name         string
	setupFile    func(string) error
	expectErr    bool
	expectedData *wrappers.KicsResultsCollection
}

func TestScanner_readKicsResultsFile(t *testing.T) {
	scanner := NewScanner(NewMockContainerManager())

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-scanner-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil && !os.IsNotExist(err) && !os.IsPermission(err) {
			t.Logf("Failed to cleanup temp dir: %v", err)
		}
	}()

	validResults := createValidKicsResults()
	emptyResults := wrappers.KicsResultsCollection{}

	tests := []readKicsTestCase{
		{
			name:         "Valid KICS results file",
			setupFile:    createValidKicsFile,
			expectErr:    false,
			expectedData: &validResults,
		},
		{
			name:         "Empty KICS results file",
			setupFile:    createEmptyKicsFile,
			expectErr:    false,
			expectedData: &emptyResults,
		},
		{
			name:         "Invalid JSON file",
			setupFile:    createInvalidKicsFile,
			expectErr:    true,
			expectedData: nil,
		},
		{
			name:         "Non-existent file",
			setupFile:    skipFileCreation,
			expectErr:    true,
			expectedData: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runReadKicsTestCase(t, scanner, tt)
		})
	}
}

// Helper function to run a single test case
func runReadKicsTestCase(t *testing.T, scanner *Scanner, testCase readKicsTestCase) {
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
	if err := testCase.setupFile(testDir); err != nil {
		t.Fatalf("Failed to setup test file: %v", err)
	}

	// Test the method
	result, err := scanner.readKicsResultsFile(testDir)

	// Validate error expectations
	if testCase.expectErr && err == nil {
		t.Error("readKicsResultsFile() expected error but got none")
		return
	}

	if !testCase.expectErr && err != nil {
		t.Errorf("readKicsResultsFile() unexpected error: %v", err)
		return
	}

	// Validate results if no error expected
	if !testCase.expectErr {
		validateKicsResults(t, &result, testCase.expectedData)
	}
}

// Test case structure for HandleScanResult
type handleScanTestCase struct {
	name       string
	err        error
	tempDir    string
	filePath   string
	expectErr  bool
	errorCheck func(error) bool
}

// Helper function to create test error based on test case name
func createTestError(testName string, originalErr error) error {
	switch testName {
	case "Engine not running error":
		return &exec.ExitError{}
	case "Invalid engine error", "Generic error":
		return &exec.Error{Name: "docker", Err: errors.New("some error")}
	default:
		return originalErr
	}
}

// Helper function to validate scan result success case
func validateScanResultSuccess(t *testing.T, results []IacRealtimeResult, err error) {
	if err != nil {
		t.Errorf("HandleScanResult() unexpected error: %v", err)
		return
	}

	if results == nil {
		t.Error("HandleScanResult() should return results on success")
	}
}

// Helper function to validate scan result error case
func validateScanResultError(t *testing.T, err error, errorCheck func(error) bool) {
	if err == nil {
		t.Error("HandleScanResult() expected error but got none")
		return
	}

	if errorCheck != nil && !errorCheck(err) {
		t.Errorf("HandleScanResult() error doesn't match expected pattern: %v", err)
	}
}

// Helper function to run a single handle scan result test case
func runHandleScanTestCase(t *testing.T, scanner *Scanner, testCase *handleScanTestCase) {
	// Create specific error types for testing
	testErr := createTestError(testCase.name, testCase.err)

	results, err := scanner.HandleScanResult(testErr, testCase.tempDir, testCase.filePath)

	if testCase.expectErr {
		validateScanResultError(t, err, testCase.errorCheck)
	} else {
		validateScanResultSuccess(t, results, err)
	}
}

// Helper function to create valid KICS results file for testing
func createValidKicsResultsFile(tempDir string) error {
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
	return os.WriteFile(filepath.Join(tempDir, ContainerResultsFileName), data, 0644)
}

func TestScanner_HandleScanResult(t *testing.T) {
	scanner := NewScanner(NewMockContainerManager())

	// Create a temp directory with valid KICS results
	tempDir, err := os.MkdirTemp("", "test-handle-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil && !os.IsNotExist(err) && !os.IsPermission(err) {
			t.Logf("Failed to cleanup temp dir: %v", err)
		}
	}()

	// Create a valid results file
	if err := createValidKicsResultsFile(tempDir); err != nil {
		t.Fatalf("Failed to create results file: %v", err)
	}

	tests := []handleScanTestCase{
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
		t.Run(tt.name, func(t *testing.T) {
			runHandleScanTestCase(t, scanner, &tt)
		})
	}
}

func TestScanner_processResults(t *testing.T) {
	scanner := NewScanner(NewMockContainerManager())

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
		setupMock func(*MockContainerManager)
	}{
		{
			name:      "Invalid engine",
			engine:    "invalid-engine",
			volumeMap: "/tmp:/path",
			tempDir:   tempDir,
			filePath:  "test.yaml",
			expectErr: true,
			setupMock: func(mock *MockContainerManager) {
				mock.ShouldFailRun = true
				mock.RunError = &exec.Error{Name: "invalid-engine", Err: errors.New("some error")}
			},
		},
		{
			name:      "Empty engine",
			engine:    "",
			volumeMap: "/tmp:/path",
			tempDir:   tempDir,
			filePath:  "test.yaml",
			expectErr: true,
			setupMock: func(mock *MockContainerManager) {
				mock.ShouldFailRun = true
				mock.RunError = &exec.Error{Name: "", Err: errors.New("some error")}
			},
		},
		{
			name:      "Valid parameters - should succeed with mock",
			engine:    "docker",
			volumeMap: "/tmp:/path",
			tempDir:   tempDir,
			filePath:  "test.yaml",
			expectErr: false,
			setupMock: func(mock *MockContainerManager) {
				// Mock succeeds by default, no setup needed
			},
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh mock for each test
			mockContainer := NewMockContainerManager()
			scanner := NewScanner(mockContainer)

			// Apply test-specific mock configuration
			ttt.setupMock(mockContainer)

			_, err := scanner.RunScan(ttt.engine, ttt.volumeMap, ttt.tempDir, ttt.filePath)

			if ttt.expectErr && err == nil {
				t.Error("RunScan() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("RunScan() unexpected error: %v", err)
			}
		})
	}
}

func TestScanner_Integration(t *testing.T) {
	// Test the full scanner workflow with mock container manager
	scanner := NewScanner(NewMockContainerManager())

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
