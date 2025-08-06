package iacrealtime

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type Mapper struct {
	lineParser *LineParser
}

func NewMapper() *Mapper {
	return &Mapper{
		lineParser: NewLineParser(),
	}
}

func (m *Mapper) ConvertKicsToIacResults(
	results *wrappers.KicsResultsCollection,
	filePath string,
) []IacRealtimeResult {
	iacResults := make([]IacRealtimeResult, 0)
	indexMap := make(map[int]LineIndex)

	data, _ := os.ReadFile(filepath.Clean(filePath))
	fileContent := string(data)

	for i := range results.Results {
		result := &results.Results[i]
		for j := range result.Locations {
			loc := &result.Locations[j]
			locLine := int(loc.Line) - 1
			lineIndex := m.getOrComputeLineIndex(fileContent, locLine, indexMap)

			iacResult := IacRealtimeResult{
				Title:        result.QueryName,
				Description:  result.Description,
				Severity:     m.mapSeverity(result.Severity),
				FilePath:     filePath,
				SimilarityID: loc.SimilarityID,
				Locations: []realtimeengine.Location{
					{
						Line:       locLine,
						StartIndex: lineIndex.Start,
						EndIndex:   lineIndex.End,
					},
				},
			}
			iacResults = append(iacResults, iacResult)
		}
	}
	return iacResults
}

func (m *Mapper) getOrComputeLineIndex(content string, lineNum int, indexMap map[int]LineIndex) LineIndex {
	if value, exists := indexMap[lineNum]; exists {
		return value
	}

	startIndex, endIndex := m.lineParser.GetLineIndices(content, lineNum)
	lineIndex := LineIndex{Start: startIndex, End: endIndex}
	indexMap[lineNum] = lineIndex
	return lineIndex
}

func (m *Mapper) mapSeverity(severity string) string {
	if mappedSeverity, exists := Severities[strings.ToLower(severity)]; exists {
		return mappedSeverity
	}
	return Severities["unknown"]
}
