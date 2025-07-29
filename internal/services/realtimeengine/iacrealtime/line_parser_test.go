package iacrealtime

import (
	"os"
	"testing"
)

func TestNewLineParser(t *testing.T) {
	parser := NewLineParser()

	if parser == nil {
		t.Error("NewLineParser() should not return nil")
	}
}

func TestLineParser_GetLineIndices(t *testing.T) {
	parser := NewLineParser()

	// Create a test file with known content
	testContent := `line 0
line 1
  line 2 with spaces  
	line 3 with tab
empty line below

line 6`

	tempFile, err := os.CreateTemp("", "test_line_parser_*.txt")
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

	tests := []struct {
		name          string
		filePath      string
		lineNumber    int
		expectedStart int
		expectedEnd   int
	}{
		{
			name:          "First line with content",
			filePath:      tempFile.Name(),
			lineNumber:    0,
			expectedStart: 0,
			expectedEnd:   5, // "line 0" - ends at '0'
		},
		{
			name:          "Second line with content",
			filePath:      tempFile.Name(),
			lineNumber:    1,
			expectedStart: 0,
			expectedEnd:   5, // "line 1" - ends at '1'
		},
		{
			name:          "Line with leading spaces",
			filePath:      tempFile.Name(),
			lineNumber:    2,
			expectedStart: 2,  // starts after "  "
			expectedEnd:   19, // ends at last 's' in "spaces"
		},
		{
			name:          "Line with tab",
			filePath:      tempFile.Name(),
			lineNumber:    3,
			expectedStart: 1,  // starts after tab
			expectedEnd:   15, // ends at 'b' in "tab"
		},
		{
			name:          "Empty line",
			filePath:      tempFile.Name(),
			lineNumber:    4,
			expectedStart: 0,
			expectedEnd:   15, // "empty line below"
		},
		{
			name:          "Line with only whitespace",
			filePath:      tempFile.Name(),
			lineNumber:    5,
			expectedStart: 0,
			expectedEnd:   0,
		},
		{
			name:          "Valid line number",
			filePath:      tempFile.Name(),
			lineNumber:    6,
			expectedStart: 0,
			expectedEnd:   5, // "line 6"
		},
		{
			name:          "Invalid line number (negative)",
			filePath:      tempFile.Name(),
			lineNumber:    -1,
			expectedStart: 0,
			expectedEnd:   0,
		},
		{
			name:          "Invalid line number (too large)",
			filePath:      tempFile.Name(),
			lineNumber:    100,
			expectedStart: 0,
			expectedEnd:   0,
		},
		{
			name:          "Empty file path",
			filePath:      "",
			lineNumber:    0,
			expectedStart: 0,
			expectedEnd:   0,
		},
		{
			name:          "Non-existent file",
			filePath:      "/nonexistent/file.txt",
			lineNumber:    0,
			expectedStart: 0,
			expectedEnd:   0,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			start, end := parser.GetLineIndices(ttt.filePath, ttt.lineNumber)
			if start != ttt.expectedStart || end != ttt.expectedEnd {
				t.Errorf("GetLineIndices() = (%d, %d), want (%d, %d)",
					start, end, ttt.expectedStart, ttt.expectedEnd)
			}
		})
	}
}

func TestLineParser_findContentBounds(t *testing.T) {
	parser := NewLineParser()

	tests := []struct {
		name          string
		line          string
		expectedStart int
		expectedEnd   int
	}{
		{
			name:          "Normal line with content",
			line:          "hello world",
			expectedStart: 0,
			expectedEnd:   10,
		},
		{
			name:          "Line with leading spaces",
			line:          "  hello world",
			expectedStart: 2,
			expectedEnd:   12,
		},
		{
			name:          "Line with trailing spaces",
			line:          "hello world  ",
			expectedStart: 0,
			expectedEnd:   10,
		},
		{
			name:          "Line with leading and trailing spaces",
			line:          "  hello world  ",
			expectedStart: 2,
			expectedEnd:   12,
		},
		{
			name:          "Line with tabs",
			line:          "\thello\tworld\t",
			expectedStart: 1,
			expectedEnd:   11,
		},
		{
			name:          "Line with mixed whitespace",
			line:          " \t hello world \t ",
			expectedStart: 3,
			expectedEnd:   13,
		},
		{
			name:          "Empty line",
			line:          "",
			expectedStart: -1,
			expectedEnd:   -1,
		},
		{
			name:          "Line with only spaces",
			line:          "   ",
			expectedStart: -1,
			expectedEnd:   -1,
		},
		{
			name:          "Line with only tabs",
			line:          "\t\t\t",
			expectedStart: -1,
			expectedEnd:   -1,
		},
		{
			name:          "Line with only mixed whitespace",
			line:          " \t \t ",
			expectedStart: -1,
			expectedEnd:   -1,
		},
		{
			name:          "Single character",
			line:          "a",
			expectedStart: 0,
			expectedEnd:   0,
		},
		{
			name:          "Single character with spaces",
			line:          "  a  ",
			expectedStart: 2,
			expectedEnd:   2,
		},
		{
			name:          "Line with carriage return",
			line:          "hello\rworld",
			expectedStart: 0,
			expectedEnd:   10,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			start, end := parser.findContentBounds(ttt.line)
			if start != ttt.expectedStart || end != ttt.expectedEnd {
				t.Errorf("findContentBounds() = (%d, %d), want (%d, %d)",
					start, end, ttt.expectedStart, ttt.expectedEnd)
			}
		})
	}
}

func TestLineParser_isWhitespace(t *testing.T) {
	parser := NewLineParser()

	tests := []struct {
		name     string
		r        rune
		expected bool
	}{
		{
			name:     "Space character",
			r:        ' ',
			expected: true,
		},
		{
			name:     "Tab character",
			r:        '\t',
			expected: true,
		},
		{
			name:     "Carriage return",
			r:        '\r',
			expected: true,
		},
		{
			name:     "Newline character",
			r:        '\n',
			expected: false, // Not considered whitespace by this function
		},
		{
			name:     "Regular letter",
			r:        'a',
			expected: false,
		},
		{
			name:     "Number",
			r:        '1',
			expected: false,
		},
		{
			name:     "Special character",
			r:        '!',
			expected: false,
		},
		{
			name:     "Unicode character",
			r:        'ñ',
			expected: false,
		},
		{
			name:     "Underscore",
			r:        '_',
			expected: false,
		},
		{
			name:     "Hyphen",
			r:        '-',
			expected: false,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := parser.isWhitespace(ttt.r)
			if got != ttt.expected {
				t.Errorf("isWhitespace() = %v, want %v", got, ttt.expected)
			}
		})
	}
}

func TestLineParser_GetLineIndices_EdgeCases(t *testing.T) {
	parser := NewLineParser()

	// Test with a file that has different line endings
	testContent := "line1\nline2\r\nline3\rline4"

	tempFile, err := os.CreateTemp("", "test_line_endings_*.txt")
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

	tests := []struct {
		name          string
		lineNumber    int
		expectedStart int
		expectedEnd   int
	}{
		{
			name:          "First line",
			lineNumber:    0,
			expectedStart: 0,
			expectedEnd:   4, // "line1"
		},
		{
			name:          "Second line with CRLF",
			lineNumber:    1,
			expectedStart: 0,
			expectedEnd:   4, // "line2"
		},
		{
			name:          "Third line with CR",
			lineNumber:    2,
			expectedStart: 0,
			expectedEnd:   10, // "line3\rline4" - ends at '4'
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			start, end := parser.GetLineIndices(tempFile.Name(), ttt.lineNumber)
			if start != ttt.expectedStart || end != ttt.expectedEnd {
				t.Errorf("GetLineIndices() = (%d, %d), want (%d, %d)",
					start, end, ttt.expectedStart, ttt.expectedEnd)
			}
		})
	}
}
