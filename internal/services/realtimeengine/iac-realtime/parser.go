package iac_realtime

import (
	"fmt"
	"os"
	"strings"
)

func getLineIndices(filePath string) (map[int]LineIndex, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	lineMap := make(map[int]LineIndex)

	for i, line := range lines {
		start := -1
		end := -1

		for j, r := range line {
			if !isWhitespace(r) {
				if start == -1 {
					start = j
				}
				end = j
			}
		}

		if start == -1 {
			start, end = 0, 0
		}

		lineMap[i] = LineIndex{
			Start: start,
			End:   end,
		}
	}

	return lineMap, nil
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r'
}
