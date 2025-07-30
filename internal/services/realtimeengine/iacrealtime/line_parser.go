package iacrealtime

import (
	"strings"
)

type LineParser struct{}

func NewLineParser() *LineParser {
	return &LineParser{}
}

func (lp *LineParser) GetLineIndices(content string, lineNumber int) (startIndex, endIndex int) {
	lines := strings.Split(content, "\n")

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
