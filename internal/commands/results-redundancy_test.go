//go:build !integration

package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a mock ScanResult with nodes for redundancy tests
func createRedundancyTestResult(id, language, queryName string, nodes []*wrappers.ScanResultNode) *wrappers.ScanResult {
	return &wrappers.ScanResult{
		ID: id,
		ScanResultData: wrappers.ScanResultData{
			LanguageName: language,
			QueryName:    queryName,
			Nodes:        nodes,
		},
	}
}

// Helper function to create a mock ScanResultNode for redundancy tests
func createRedundancyTestNode(fileName string, line, column uint) *wrappers.ScanResultNode {
	return &wrappers.ScanResultNode{
		FileName: fileName,
		Line:     line,
		Column:   column,
	}
}

// TestGetLanguages tests the GetLanguages function
func TestGetLanguages(t *testing.T) {
	tests := []struct {
		name     string
		results  *wrappers.ScanResultsCollection
		expected map[string]bool
	}{
		{
			name:     "Empty results",
			results:  &wrappers.ScanResultsCollection{Results: []*wrappers.ScanResult{}},
			expected: map[string]bool{},
		},
		{
			name: "Single language",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "query1", nil),
				},
			},
			expected: map[string]bool{"Java": true},
		},
		{
			name: "Multiple languages",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "query1", nil),
					createRedundancyTestResult("2", "Python", "query2", nil),
					createRedundancyTestResult("3", "JavaScript", "query3", nil),
				},
			},
			expected: map[string]bool{"Java": true, "Python": true, "JavaScript": true},
		},
		{
			name: "Duplicate languages",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "query1", nil),
					createRedundancyTestResult("2", "Java", "query2", nil),
					createRedundancyTestResult("3", "Python", "query3", nil),
				},
			},
			expected: map[string]bool{"Java": true, "Python": true},
		},
		{
			name: "Empty language name skipped",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "query1", nil),
					createRedundancyTestResult("2", "", "query2", nil),
				},
			},
			expected: map[string]bool{"Java": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLanguages(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetQueries tests the GetQueries function
func TestGetQueries(t *testing.T) {
	tests := []struct {
		name      string
		results   *wrappers.ScanResultsCollection
		languages map[string]bool
		expected  map[string]map[string]bool
	}{
		{
			name:      "Empty results",
			results:   &wrappers.ScanResultsCollection{Results: []*wrappers.ScanResult{}},
			languages: map[string]bool{"Java": true},
			expected:  map[string]map[string]bool{},
		},
		{
			name: "Single language single query",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "SQL_Injection", nil),
				},
			},
			languages: map[string]bool{"Java": true},
			expected:  map[string]map[string]bool{"Java": {"SQL_Injection": true}},
		},
		{
			name: "Multiple queries per language",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "SQL_Injection", nil),
					createRedundancyTestResult("2", "Java", "XSS", nil),
					createRedundancyTestResult("3", "Python", "Command_Injection", nil),
				},
			},
			languages: map[string]bool{"Java": true, "Python": true},
			expected: map[string]map[string]bool{
				"Java":   {"SQL_Injection": true, "XSS": true},
				"Python": {"Command_Injection": true},
			},
		},
		{
			name: "Language not in filter skipped",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "SQL_Injection", nil),
					createRedundancyTestResult("2", "Ruby", "XSS", nil),
				},
			},
			languages: map[string]bool{"Java": true},
			expected:  map[string]map[string]bool{"Java": {"SQL_Injection": true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetQueries(tt.results, tt.languages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetKey tests the GetKey function
func TestGetKey(t *testing.T) {
	tests := []struct {
		name     string
		r1       string
		r2       string
		expected string
	}{
		{
			name:     "r1 less than r2",
			r1:       "a",
			r2:       "b",
			expected: "a,b",
		},
		{
			name:     "r2 less than r1",
			r1:       "b",
			r2:       "a",
			expected: "a,b",
		},
		{
			name:     "Same values",
			r1:       "same",
			r2:       "same",
			expected: "same,same",
		},
		{
			name:     "Numeric strings",
			r1:       "123",
			r2:       "456",
			expected: "123,456",
		},
		{
			name:     "Empty strings",
			r1:       "",
			r2:       "a",
			expected: ",a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetKey(tt.r1, tt.r2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRoundFloat tests the roundFloat function
func TestRoundFloat(t *testing.T) {
	tests := []struct {
		name      string
		val       float64
		precision uint
		expected  float64
	}{
		{
			name:      "Round down",
			val:       1.234,
			precision: 2,
			expected:  1.23,
		},
		{
			name:      "Round up",
			val:       1.235,
			precision: 2,
			expected:  1.24,
		},
		{
			name:      "No decimal places",
			val:       1.5,
			precision: 0,
			expected:  2.0,
		},
		{
			name:      "Already rounded",
			val:       1.0,
			precision: 2,
			expected:  1.0,
		},
		{
			name:      "High precision",
			val:       1.23456789,
			precision: 4,
			expected:  1.2346,
		},
		{
			name:      "Zero value",
			val:       0.0,
			precision: 2,
			expected:  0.0,
		},
		{
			name:      "Negative value",
			val:       -1.234,
			precision: 2,
			expected:  -1.23,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roundFloat(tt.val, tt.precision)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetSha1String tests the getSha1String function
func TestGetSha1String(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Single line",
			lines: []string{"hello"},
		},
		{
			name:  "Multiple lines",
			lines: []string{"hello", "world"},
		},
		{
			name:  "Empty slice",
			lines: []string{},
		},
		{
			name:  "Lines with special characters",
			lines: []string{"file.go:10:5", "file.go:20:10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSha1String(tt.lines)
			assert.NotEmpty(t, result)
			// Verify deterministic output
			result2 := getSha1String(tt.lines)
			assert.Equal(t, result, result2)
			// SHA1 produces 40 character hex string
			assert.Len(t, result, 40)
		})
	}

	// Test that different inputs produce different outputs
	t.Run("Different inputs produce different outputs", func(t *testing.T) {
		result1 := getSha1String([]string{"a", "b"})
		result2 := getSha1String([]string{"c", "d"})
		assert.NotEqual(t, result1, result2)
	})
}

// TestGetResultsForQuery tests the GetResultsForQuery function
func TestGetResultsForQuery(t *testing.T) {
	tests := []struct {
		name        string
		results     *wrappers.ScanResultsCollection
		language    string
		query       string
		expectedLen int
		expectedIDs []string
	}{
		{
			name:        "Empty results",
			results:     &wrappers.ScanResultsCollection{Results: []*wrappers.ScanResult{}},
			language:    "Java",
			query:       "SQL_Injection",
			expectedLen: 0,
			expectedIDs: nil,
		},
		{
			name: "Single matching result",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "SQL_Injection", nil),
				},
			},
			language:    "Java",
			query:       "SQL_Injection",
			expectedLen: 1,
			expectedIDs: []string{"1"},
		},
		{
			name: "Multiple matching results sorted by ID",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("3", "Java", "SQL_Injection", nil),
					createRedundancyTestResult("1", "Java", "SQL_Injection", nil),
					createRedundancyTestResult("2", "Java", "SQL_Injection", nil),
				},
			},
			language:    "Java",
			query:       "SQL_Injection",
			expectedLen: 3,
			expectedIDs: []string{"1", "2", "3"},
		},
		{
			name: "Filter by language",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "SQL_Injection", nil),
					createRedundancyTestResult("2", "Python", "SQL_Injection", nil),
				},
			},
			language:    "Java",
			query:       "SQL_Injection",
			expectedLen: 1,
			expectedIDs: []string{"1"},
		},
		{
			name: "Filter by query",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "SQL_Injection", nil),
					createRedundancyTestResult("2", "Java", "XSS", nil),
				},
			},
			language:    "Java",
			query:       "SQL_Injection",
			expectedLen: 1,
			expectedIDs: []string{"1"},
		},
		{
			name: "No matching results",
			results: &wrappers.ScanResultsCollection{
				Results: []*wrappers.ScanResult{
					createRedundancyTestResult("1", "Java", "XSS", nil),
				},
			},
			language:    "Java",
			query:       "SQL_Injection",
			expectedLen: 0,
			expectedIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetResultsForQuery(tt.results, tt.language, tt.query)
			assert.Len(t, result, tt.expectedLen)
			if tt.expectedIDs != nil {
				for i, expectedID := range tt.expectedIDs {
					assert.Equal(t, expectedID, result[i].ID)
				}
			}
		})
	}
}

// TestGetResultForID tests the getResultForID function
func TestGetResultForID(t *testing.T) {
	results := []*wrappers.ScanResult{
		createRedundancyTestResult("1", "Java", "query1", nil),
		createRedundancyTestResult("2", "Python", "query2", nil),
		createRedundancyTestResult("3", "JavaScript", "query3", nil),
	}

	tests := []struct {
		name     string
		resultID string
		expected *wrappers.ScanResult
	}{
		{
			name:     "Find existing result",
			resultID: "2",
			expected: results[1],
		},
		{
			name:     "Result not found",
			resultID: "999",
			expected: nil,
		},
		{
			name:     "Empty ID",
			resultID: "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getResultForID(results, tt.resultID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildFlows tests the buildFlows function
func TestBuildFlows(t *testing.T) {
	tests := []struct {
		name         string
		queryResults []*wrappers.ScanResult
		expectedKeys []string
	}{
		{
			name:         "Empty results",
			queryResults: []*wrappers.ScanResult{},
			expectedKeys: []string{},
		},
		{
			name: "Single result with nodes",
			queryResults: []*wrappers.ScanResult{
				createRedundancyTestResult("1", "Java", "query", []*wrappers.ScanResultNode{
					createRedundancyTestNode("file.java", 10, 5),
					createRedundancyTestNode("file.java", 20, 10),
				}),
			},
			expectedKeys: []string{"1"},
		},
		{
			name: "Multiple results",
			queryResults: []*wrappers.ScanResult{
				createRedundancyTestResult("1", "Java", "query", []*wrappers.ScanResultNode{
					createRedundancyTestNode("file.java", 10, 5),
				}),
				createRedundancyTestResult("2", "Java", "query", []*wrappers.ScanResultNode{
					createRedundancyTestNode("file.java", 20, 10),
				}),
			},
			expectedKeys: []string{"1", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFlows(tt.queryResults)
			assert.Len(t, result, len(tt.expectedKeys))
			for _, key := range tt.expectedKeys {
				_, exists := result[key]
				assert.True(t, exists, "Expected key %s to exist", key)
			}
		})
	}
}

// TestBuildFlows_NodeFormat tests that buildFlows creates correct node string format
func TestBuildFlows_NodeFormat(t *testing.T) {
	queryResults := []*wrappers.ScanResult{
		createRedundancyTestResult("1", "Java", "query", []*wrappers.ScanResultNode{
			createRedundancyTestNode("file.java", 10, 5),
			createRedundancyTestNode("file.java", 20, 10),
		}),
	}

	result := buildFlows(queryResults)
	flows := result["1"]

	assert.Len(t, flows, 2)
	assert.Equal(t, "file.java:10:5", flows[0])
	assert.Equal(t, "file.java:20:10", flows[1])
}

// TestSortSubFlowIDs tests the sortSubFlowIDs function
func TestSortSubFlowIDs(t *testing.T) {
	tests := []struct {
		name     string
		subFlows map[string]*SubFlow
		expected []string
	}{
		{
			name:     "Empty map",
			subFlows: map[string]*SubFlow{},
			expected: nil,
		},
		{
			name: "Single subflow",
			subFlows: map[string]*SubFlow{
				"abc": {ShaOne: "abc"},
			},
			expected: []string{"abc"},
		},
		{
			name: "Multiple subflows sorted",
			subFlows: map[string]*SubFlow{
				"zzz": {ShaOne: "zzz"},
				"aaa": {ShaOne: "aaa"},
				"mmm": {ShaOne: "mmm"},
			},
			expected: []string{"aaa", "mmm", "zzz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortSubFlowIDs(tt.subFlows)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestSortSubFlowResultIDs tests the sortSubFlowResultIDs function
func TestSortSubFlowResultIDs(t *testing.T) {
	tests := []struct {
		name     string
		subFlow  *SubFlow
		expected []string
	}{
		{
			name: "Empty results",
			subFlow: &SubFlow{
				Results: map[string]bool{},
			},
			expected: nil,
		},
		{
			name: "Single result",
			subFlow: &SubFlow{
				Results: map[string]bool{"1": true},
			},
			expected: []string{"1"},
		},
		{
			name: "Multiple results sorted",
			subFlow: &SubFlow{
				Results: map[string]bool{
					"3": true,
					"1": true,
					"2": true,
				},
			},
			expected: []string{"1", "2", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortSubFlowResultIDs(tt.subFlow)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestGetMaxCoverage tests the getMaxCoverage function
func TestGetMaxCoverage(t *testing.T) {
	tests := []struct {
		name                string
		coverage            map[string]float64
		expectedID          string
		expectedMaxCoverage float64
	}{
		{
			name:                "Empty coverage",
			coverage:            map[string]float64{},
			expectedID:          "",
			expectedMaxCoverage: 0.0,
		},
		{
			name:                "Single entry",
			coverage:            map[string]float64{"1": 0.75},
			expectedID:          "1",
			expectedMaxCoverage: 0.75,
		},
		{
			name:                "Multiple entries",
			coverage:            map[string]float64{"1": 0.5, "2": 0.8, "3": 0.6},
			expectedID:          "2",
			expectedMaxCoverage: 0.8,
		},
		{
			name:                "Tie goes to lexically first",
			coverage:            map[string]float64{"b": 0.5, "a": 0.5},
			expectedID:          "a",
			expectedMaxCoverage: 0.5,
		},
		{
			name:                "All zero coverage",
			coverage:            map[string]float64{"1": 0.0, "2": 0.0},
			expectedID:          "",
			expectedMaxCoverage: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultID, maxCov := getMaxCoverage(tt.coverage)
			assert.Equal(t, tt.expectedID, resultID)
			assert.Equal(t, tt.expectedMaxCoverage, maxCov)
		})
	}
}

// TestComputeSubFlow tests the computeSubFlow function
func TestComputeSubFlow(t *testing.T) {
	tests := []struct {
		name         string
		f1           []string
		f2           []string
		expectExists bool
		expectedFlow []string
	}{
		{
			name:         "No common elements",
			f1:           []string{"a", "b", "c"},
			f2:           []string{"d", "e", "f"},
			expectExists: false,
			expectedFlow: nil,
		},
		{
			name:         "Exact match",
			f1:           []string{"a", "b", "c"},
			f2:           []string{"a", "b", "c"},
			expectExists: true,
			expectedFlow: []string{"a", "b", "c"},
		},
		{
			name:         "Partial match at start",
			f1:           []string{"a", "b", "c"},
			f2:           []string{"a", "b", "d"},
			expectExists: true,
			expectedFlow: []string{"a", "b"},
		},
		{
			name:         "Common subsequence in middle",
			f1:           []string{"x", "a", "b", "y"},
			f2:           []string{"z", "a", "b", "w"},
			expectExists: true,
			expectedFlow: []string{"a", "b"},
		},
		{
			name:         "Single common element",
			f1:           []string{"a", "b", "c"},
			f2:           []string{"x", "a", "y"},
			expectExists: true,
			expectedFlow: []string{"a"},
		},
		{
			name:         "Empty first flow",
			f1:           []string{},
			f2:           []string{"a", "b"},
			expectExists: false,
			expectedFlow: nil,
		},
		{
			name:         "Empty second flow",
			f1:           []string{"a", "b"},
			f2:           []string{},
			expectExists: false,
			expectedFlow: nil,
		},
		{
			name:         "Both flows empty",
			f1:           []string{},
			f2:           []string{},
			expectExists: false,
			expectedFlow: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, subFlow := computeSubFlow(tt.f1, tt.f2)
			assert.Equal(t, tt.expectExists, exists)
			if tt.expectExists {
				assert.NotNil(t, subFlow)
				assert.Equal(t, tt.expectedFlow, subFlow.Flow)
				assert.NotEmpty(t, subFlow.ShaOne)
			} else {
				assert.Nil(t, subFlow)
			}
		})
	}
}

// TestCompareFlows tests the compareFlows function
func TestCompareFlows(t *testing.T) {
	tests := []struct {
		name     string
		flows    map[string][]string
		expected int // number of subflows expected
	}{
		{
			name:     "Empty flows",
			flows:    map[string][]string{},
			expected: 0,
		},
		{
			name: "Single flow",
			flows: map[string][]string{
				"1": {"a", "b", "c"},
			},
			expected: 0,
		},
		{
			name: "Two flows with common subflow",
			flows: map[string][]string{
				"1": {"a", "b", "c"},
				"2": {"a", "b", "d"},
			},
			expected: 1,
		},
		{
			name: "Two flows with no common subflow",
			flows: map[string][]string{
				"1": {"a", "b", "c"},
				"2": {"d", "e", "f"},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareFlows(tt.flows)
			assert.Len(t, result, tt.expected)
		})
	}
}

// TestLabelRedundantResults tests the labelRedundantResults function
func TestLabelRedundantResults(t *testing.T) {
	t.Run("Labels fix for results with no redundant entries", func(t *testing.T) {
		results := []*wrappers.ScanResult{
			createRedundancyTestResult("1", "Java", "query", nil),
		}
		redundantResults := map[string]map[string]bool{
			"1": {}, // Empty map means no redundancy
		}
		labelRedundantResults(results, redundantResults)
		assert.Equal(t, "fix", results[0].ScanResultData.Redundancy)
	})

	t.Run("Labels redundant for results with redundant entries", func(t *testing.T) {
		results := []*wrappers.ScanResult{
			createRedundancyTestResult("1", "Java", "query", nil),
			createRedundancyTestResult("2", "Java", "query", nil),
		}
		redundantResults := map[string]map[string]bool{
			"1": {},          // This is the fix
			"2": {"1": true}, // This is redundant to 1
		}
		labelRedundantResults(results, redundantResults)
		assert.Equal(t, "fix", results[0].ScanResultData.Redundancy)
		assert.Equal(t, "redundant", results[1].ScanResultData.Redundancy)
	})
}

// TestComputeRedundantResults tests the computeRedundantResults function
func TestComputeRedundantResults(t *testing.T) {
	t.Run("Empty subflows returns all empty redundant maps", func(t *testing.T) {
		results := []*wrappers.ScanResult{
			createRedundancyTestResult("1", "Java", "query", []*wrappers.ScanResultNode{
				createRedundancyTestNode("file.java", 10, 5),
			}),
		}
		subFlows := map[string]*SubFlow{}

		redundant := computeRedundantResults(subFlows, results)

		assert.Len(t, redundant, 1)
		assert.Len(t, redundant["1"], 0)
	})

	t.Run("High coverage marks redundancy", func(t *testing.T) {
		// Result 1 has 2 nodes, Result 2 has 4 nodes
		// Subflow has 2 nodes
		// Coverage for 1 = 2/2 = 1.0 (100%)
		// Coverage for 2 = 2/4 = 0.5 (50%)
		// Since result 1 has higher coverage, result 2 is redundant to 1
		results := []*wrappers.ScanResult{
			createRedundancyTestResult("1", "Java", "query", []*wrappers.ScanResultNode{
				createRedundancyTestNode("file.java", 10, 5),
				createRedundancyTestNode("file.java", 20, 10),
			}),
			createRedundancyTestResult("2", "Java", "query", []*wrappers.ScanResultNode{
				createRedundancyTestNode("file.java", 10, 5),
				createRedundancyTestNode("file.java", 20, 10),
				createRedundancyTestNode("file.java", 30, 15),
				createRedundancyTestNode("file.java", 40, 20),
			}),
		}
		subFlows := map[string]*SubFlow{
			"sha1": {
				ShaOne: "sha1",
				Flow:   []string{"file.java:10:5", "file.java:20:10"},
				Results: map[string]bool{
					"1": true,
					"2": true,
				},
			},
		}

		redundant := computeRedundantResults(subFlows, results)

		assert.Len(t, redundant, 2)
		// Result 1 has 100% coverage, so result 2 is redundant to it
		assert.Len(t, redundant["1"], 0)
		assert.True(t, redundant["2"]["1"])
	})

	t.Run("Low coverage skipped", func(t *testing.T) {
		// Both results have 4 nodes, subflow has 1 node
		// Coverage = 1/4 = 0.25 (below 0.5 threshold)
		results := []*wrappers.ScanResult{
			createRedundancyTestResult("1", "Java", "query", []*wrappers.ScanResultNode{
				createRedundancyTestNode("file.java", 10, 5),
				createRedundancyTestNode("file.java", 20, 10),
				createRedundancyTestNode("file.java", 30, 15),
				createRedundancyTestNode("file.java", 40, 20),
			}),
			createRedundancyTestResult("2", "Java", "query", []*wrappers.ScanResultNode{
				createRedundancyTestNode("file.java", 10, 5),
				createRedundancyTestNode("file.java", 50, 25),
				createRedundancyTestNode("file.java", 60, 30),
				createRedundancyTestNode("file.java", 70, 35),
			}),
		}
		subFlows := map[string]*SubFlow{
			"sha1": {
				ShaOne:  "sha1",
				Flow:    []string{"file.java:10:5"},
				Results: map[string]bool{"1": true, "2": true},
			},
		}

		redundant := computeRedundantResults(subFlows, results)

		// Both should have empty redundancy since coverage < 0.5
		assert.Len(t, redundant["1"], 0)
		assert.Len(t, redundant["2"], 0)
	})
}

// TestComputeRedundantSastResultsForQuery tests the full query flow
func TestComputeRedundantSastResultsForQuery(t *testing.T) {
	t.Run("Empty results returns unchanged", func(t *testing.T) {
		results := &wrappers.ScanResultsCollection{Results: []*wrappers.ScanResult{}}
		result := ComputeRedundantSastResultsForQuery(results, "Java", "query")
		assert.Equal(t, results, result)
	})

	t.Run("No matching query returns unchanged", func(t *testing.T) {
		results := &wrappers.ScanResultsCollection{
			Results: []*wrappers.ScanResult{
				createRedundancyTestResult("1", "Java", "OtherQuery", nil),
			},
		}
		result := ComputeRedundantSastResultsForQuery(results, "Java", "SQL_Injection")
		assert.Equal(t, results, result)
	})
}

// TestComputeRedundantSastResults tests the main entry point
func TestComputeRedundantSastResults(t *testing.T) {
	t.Run("Empty results", func(t *testing.T) {
		results := &wrappers.ScanResultsCollection{Results: []*wrappers.ScanResult{}}
		result := ComputeRedundantSastResults(results)
		assert.NotNil(t, result)
		assert.Len(t, result.Results, 0)
	})

	t.Run("Single result without redundancy", func(t *testing.T) {
		results := &wrappers.ScanResultsCollection{
			Results: []*wrappers.ScanResult{
				createRedundancyTestResult("1", "Java", "SQL_Injection", []*wrappers.ScanResultNode{
					createRedundancyTestNode("file.java", 10, 5),
				}),
			},
		}
		result := ComputeRedundantSastResults(results)
		assert.NotNil(t, result)
		assert.Len(t, result.Results, 1)
	})

	t.Run("Multiple languages processed", func(t *testing.T) {
		results := &wrappers.ScanResultsCollection{
			Results: []*wrappers.ScanResult{
				createRedundancyTestResult("1", "Java", "SQL_Injection", []*wrappers.ScanResultNode{
					createRedundancyTestNode("file.java", 10, 5),
				}),
				createRedundancyTestResult("2", "Python", "SQL_Injection", []*wrappers.ScanResultNode{
					createRedundancyTestNode("file.py", 20, 10),
				}),
			},
		}
		result := ComputeRedundantSastResults(results)
		assert.NotNil(t, result)
		assert.Len(t, result.Results, 2)
	})
}
