package iacrealtime

import (
	"os"
	"path/filepath"
	"strings"
)

type LineParser struct{}

func NewLineParser() *LineParser {
	return &LineParser{}
}

func (lp *LineParser) GetLineIndices(filePath string, lineNumber int) (startIndex int, endIndex int) {
	if filePath == "" {
		return 0, 0
	}

	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return 0, 0
	}

	lines := strings.Split(string(data), "\n")

	if lineNumber < 0 || lineNumber >= len(lines) {
		return 0, 0
	}

	line := lines[lineNumber]
	start, end := lp.findContentBounds(line)

	if start == -1 {
		// Line is empty or only whitespace
		return 0, 0
	}

	return start, end
}

func (lp *LineParser) findContentBounds(line string) (start, end int) {
	start = -1
	end = -1

	for i, r := range line {
		if !lp.isWhitespace(r) {
			if start == -1 {
				start = i
			}
			end = i
		}
	}

	return start, end
}

func (lp *LineParser) isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r'
}
