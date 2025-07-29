package iacrealtime

import (
	"os"
	"reflect"
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

func TestNewMapper(t *testing.T) {
	mapper := NewMapper()

	if mapper == nil {
		t.Error("NewMapper() should not return nil")
	}

	if mapper.lineParser == nil {
		t.Error("NewMapper() should initialize lineParser")
	}
}

func TestMapper_ConvertKicsToIacResults(t *testing.T) {
	mapper := NewMapper()

	type args struct {
		results  *wrappers.KicsResultsCollection
		filePath string
	}
	tests := []struct {
		name     string
		args     args
		expected []IacRealtimeResult
	}{
		{
			name: "Convert empty results collection",
			args: args{
				results: &wrappers.KicsResultsCollection{
					Results: []wrappers.KicsQueries{},
				},
				filePath: "test.yaml",
			},
			expected: []IacRealtimeResult{},
		},
		{
			name: "Convert single result with one location",
			args: args{
				results: &wrappers.KicsResultsCollection{
					Results: []wrappers.KicsQueries{
						{
							QueryName:   "Test Query",
							Description: "Test Description",
							Severity:    "high",
							Locations: []wrappers.KicsFiles{
								{
									SimilarityID: "test-similarity-id",
									Line:         10,
								},
							},
						},
					},
				},
				filePath: "test.yaml",
			},
			expected: []IacRealtimeResult{
				{
					Title:        "Test Query",
					Description:  "Test Description",
					Severity:     "High",
					FilePath:     "test.yaml",
					SimilarityID: "test-similarity-id",
					Locations: []realtimeengine.Location{
						{
							Line:       9, // Line is 0-indexed in the result
							StartIndex: 0,
							EndIndex:   0,
						},
					},
				},
			},
		},
		{
			name: "Convert result with multiple locations",
			args: args{
				results: &wrappers.KicsResultsCollection{
					Results: []wrappers.KicsQueries{
						{
							QueryName:   "Multi Location Query",
							Description: "Test with multiple locations",
							Severity:    "medium",
							Locations: []wrappers.KicsFiles{
								{
									SimilarityID: "similarity-1",
									Line:         5,
								},
								{
									SimilarityID: "similarity-2",
									Line:         15,
								},
							},
						},
					},
				},
				filePath: "test.yaml",
			},
			expected: []IacRealtimeResult{
				{
					Title:        "Multi Location Query",
					Description:  "Test with multiple locations",
					Severity:     "Medium",
					FilePath:     "test.yaml",
					SimilarityID: "similarity-1",
					Locations: []realtimeengine.Location{
						{
							Line:       4,
							StartIndex: 0,
							EndIndex:   0,
						},
					},
				},
				{
					Title:        "Multi Location Query",
					Description:  "Test with multiple locations",
					Severity:     "Medium",
					FilePath:     "test.yaml",
					SimilarityID: "similarity-2",
					Locations: []realtimeengine.Location{
						{
							Line:       14,
							StartIndex: 0,
							EndIndex:   0,
						},
					},
				},
			},
		},
		{
			name: "Convert multiple results",
			args: args{
				results: &wrappers.KicsResultsCollection{
					Results: []wrappers.KicsQueries{
						{
							QueryName:   "Query 1",
							Description: "Description 1",
							Severity:    "critical",
							Locations: []wrappers.KicsFiles{
								{
									SimilarityID: "sim-1",
									Line:         1,
								},
							},
						},
						{
							QueryName:   "Query 2",
							Description: "Description 2",
							Severity:    "low",
							Locations: []wrappers.KicsFiles{
								{
									SimilarityID: "sim-2",
									Line:         2,
								},
							},
						},
					},
				},
				filePath: "test.yaml",
			},
			expected: []IacRealtimeResult{
				{
					Title:        "Query 1",
					Description:  "Description 1",
					Severity:     "Critical",
					FilePath:     "test.yaml",
					SimilarityID: "sim-1",
					Locations: []realtimeengine.Location{
						{
							Line:       0,
							StartIndex: 0,
							EndIndex:   0,
						},
					},
				},
				{
					Title:        "Query 2",
					Description:  "Description 2",
					Severity:     "Low",
					FilePath:     "test.yaml",
					SimilarityID: "sim-2",
					Locations: []realtimeengine.Location{
						{
							Line:       1,
							StartIndex: 0,
							EndIndex:   0,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := mapper.ConvertKicsToIacResults(ttt.args.results, ttt.args.filePath)
			if !reflect.DeepEqual(got, ttt.expected) {
				t.Errorf("ConvertKicsToIacResults() = %v, want %v", got, ttt.expected)
			}
		})
	}
}

func TestMapper_mapSeverity(t *testing.T) {
	mapper := NewMapper()

	tests := []struct {
		name     string
		severity string
		expected string
	}{
		{
			name:     "Map critical severity",
			severity: "critical",
			expected: "Critical",
		},
		{
			name:     "Map high severity",
			severity: "high",
			expected: "High",
		},
		{
			name:     "Map medium severity",
			severity: "medium",
			expected: "Medium",
		},
		{
			name:     "Map low severity",
			severity: "low",
			expected: "Low",
		},
		{
			name:     "Map info severity",
			severity: "info",
			expected: "Info",
		},
		{
			name:     "Map uppercase severity",
			severity: "HIGH",
			expected: "High",
		},
		{
			name:     "Map mixed case severity",
			severity: "CrItIcAl",
			expected: "Critical",
		},
		{
			name:     "Map unknown severity",
			severity: "invalid",
			expected: "Unknown",
		},
		{
			name:     "Map empty severity",
			severity: "",
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := mapper.mapSeverity(ttt.severity)
			if got != ttt.expected {
				t.Errorf("mapSeverity() = %v, want %v", got, ttt.expected)
			}
		})
	}
}

func TestMapper_getOrComputeLineIndex(t *testing.T) {
	mapper := NewMapper()

	// Create a temporary file for testing
	content := "line 0\nline 1\n  line 2 with spaces  \nline 3\n"
	tempFile, err := os.CreateTemp("", "test_file_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	tests := []struct {
		name         string
		filePath     string
		lineNum      int
		indexMap     map[int]LineIndex
		expected     LineIndex
		expectCached bool
	}{
		{
			name:     "Compute line index for valid line",
			filePath: tempFile.Name(),
			lineNum:  2,
			indexMap: make(map[int]LineIndex),
			expected: LineIndex{Start: 2, End: 19}, // "  line 2 with spaces  " - starts at index 2, ends at index 19 (last 's')
		},
		{
			name:     "Return cached line index",
			filePath: tempFile.Name(),
			lineNum:  1,
			indexMap: map[int]LineIndex{
				1: {Start: 10, End: 20},
			},
			expected:     LineIndex{Start: 10, End: 20},
			expectCached: true,
		},
		{
			name:     "Handle invalid file path",
			filePath: "/nonexistent/file.txt",
			lineNum:  0,
			indexMap: make(map[int]LineIndex),
			expected: LineIndex{Start: 0, End: 0},
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := mapper.getOrComputeLineIndex(ttt.filePath, ttt.lineNum, ttt.indexMap)
			if !reflect.DeepEqual(got, ttt.expected) {
				t.Errorf("getOrComputeLineIndex() = %v, want %v", got, ttt.expected)
			}

			// Check if the value was cached (if not already in map)
			if !ttt.expectCached {
				if cachedValue, exists := ttt.indexMap[ttt.lineNum]; !exists || !reflect.DeepEqual(cachedValue, ttt.expected) {
					t.Errorf("getOrComputeLineIndex() should cache the computed value")
				}
			}
		})
	}
}

func TestMapper_getOrComputeLineIndex_Integration(t *testing.T) {
	mapper := NewMapper()

	// Create a test file with known content
	testContent := `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: test`

	tempFile, err := os.CreateTemp("", "test_integration_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	if _, err := tempFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Test that the same line number returns cached results on subsequent calls
	indexMap := make(map[int]LineIndex)

	// First call should compute
	result1 := mapper.getOrComputeLineIndex(tempFile.Name(), 1, indexMap)

	// Second call should use cache
	result2 := mapper.getOrComputeLineIndex(tempFile.Name(), 1, indexMap)

	if !reflect.DeepEqual(result1, result2) {
		t.Errorf("Cached result should be identical to computed result")
	}

	if len(indexMap) != 1 {
		t.Errorf("Expected indexMap to have 1 entry, got %d", len(indexMap))
	}
}

func TestMapper_ConvertKicsToIacResults_WithRealFile(t *testing.T) {
	mapper := NewMapper()

	// Create a real file for testing line index computation
	testContent := `apiVersion: v1
kind: Pod
metadata:
  name: test-pod`

	tempFile, err := os.CreateTemp("", "test_real_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	if _, err := tempFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	results := &wrappers.KicsResultsCollection{
		Results: []wrappers.KicsQueries{
			{
				QueryName:   "Test Query",
				Description: "Test Description",
				Severity:    "high",
				Locations: []wrappers.KicsFiles{
					{
						SimilarityID: "test-sim-id",
						Line:         2, // "kind: Pod" line
					},
				},
			},
		},
	}

	iacResults := mapper.ConvertKicsToIacResults(results, tempFile.Name())

	if len(iacResults) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(iacResults))
	}

	result := iacResults[0]
	if result.Title != "Test Query" {
		t.Errorf("Expected Title 'Test Query', got '%s'", result.Title)
	}

	if result.Severity != "High" {
		t.Errorf("Expected Severity 'High', got '%s'", result.Severity)
	}

	if result.FilePath != tempFile.Name() {
		t.Errorf("Expected FilePath '%s', got '%s'", tempFile.Name(), result.FilePath)
	}

	if len(result.Locations) != 1 {
		t.Fatalf("Expected 1 location, got %d", len(result.Locations))
	}

	location := result.Locations[0]
	if location.Line != 1 { // Line 2 becomes 1 (0-indexed)
		t.Errorf("Expected Line 1, got %d", location.Line)
	}

	// The "kind: Pod" line should start at index 0 and end at index 8 (last 'd')
	if location.StartIndex != 0 || location.EndIndex != 8 {
		t.Errorf("Expected StartIndex 0 and EndIndex 8, got StartIndex %d and EndIndex %d",
			location.StartIndex, location.EndIndex)
	}
}
