package iacrealtime

import (
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
	var iacResults []IacRealtimeResult
	indexMap := make(map[int]LineIndex)

	for i := range results.Results {
		result := &results.Results[i]
		for j := range result.Locations {
			loc := &result.Locations[j]
			locLine := int(loc.Line) - 1
			lineIndex := m.getOrComputeLineIndex(filePath, locLine, indexMap)

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

func (m *Mapper) getOrComputeLineIndex(filePath string, lineNum int, indexMap map[int]LineIndex) LineIndex {
	if value, exists := indexMap[lineNum]; exists {
		return value
	}

	startIndex, endIndex := m.lineParser.GetLineIndices(filePath, lineNum)
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
