package commands

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	precision = 2
	ten       = 10
	half      = 0.5
)

func ComputeRedundantSastResults(resultsModel *wrappers.ScanResultsCollection) *wrappers.ScanResultsCollection {
	languages := GetLanguages(resultsModel)
	queriesByLanguage := GetQueries(resultsModel, languages)

	for language := range queriesByLanguage {
		for query := range queriesByLanguage[language] {
			resultsModel = ComputeRedundantSastResultsForQuery(resultsModel, language, query)
		}
	}
	return resultsModel
}

func GetLanguages(resultsModel *wrappers.ScanResultsCollection) map[string]bool {
	languages := make(map[string]bool)
	for _, result := range resultsModel.Results {
		if result.ScanResultData.LanguageName != "" {
			languages[result.ScanResultData.LanguageName] = true
		}
	}
	return languages
}

func GetQueries(resultsModel *wrappers.ScanResultsCollection, languages map[string]bool) map[string]map[string]bool {
	queriesByLanguage := make(map[string]map[string]bool)
	for _, result := range resultsModel.Results {
		if _, exist := languages[result.ScanResultData.LanguageName]; !exist {
			continue
		}
		if _, exist := queriesByLanguage[result.ScanResultData.LanguageName]; !exist {
			queriesByLanguage[result.ScanResultData.LanguageName] = make(map[string]bool)
		}
		queriesByLanguage[result.ScanResultData.LanguageName][result.ScanResultData.QueryName] = true
	}
	return queriesByLanguage
}

func ComputeRedundantSastResultsForQuery(resultsModel *wrappers.ScanResultsCollection, language, query string) *wrappers.ScanResultsCollection {
	queryResults := GetResultsForQuery(resultsModel, language, query)
	if len(queryResults) == 0 {
		return resultsModel
	}

	flows := buildFlows(queryResults)

	subFlows := compareFlows(flows)

	redundantResults := computeRedundantResults(subFlows, queryResults)

	labelRedundantResults(queryResults, redundantResults)
	return resultsModel
}

func GetResultsForQuery(resultsModel *wrappers.ScanResultsCollection, language, query string) []*wrappers.ScanResult {
	var queryResults []*wrappers.ScanResult
	for _, result := range resultsModel.Results {
		if result.ScanResultData.LanguageName != language || result.ScanResultData.QueryName != query {
			continue
		}
		queryResults = append(queryResults, result)
	}

	sort.Slice(queryResults, func(i, j int) bool {
		return queryResults[i].ID < queryResults[j].ID
	})

	return queryResults
}

func labelRedundantResults(queryResults []*wrappers.ScanResult, redundantResults map[string]map[string]bool) {
	resultsByID := make(map[string]*wrappers.ScanResult)
	for _, result := range queryResults {
		resultsByID[result.ID] = result
	}

	for resultID, redundantResult := range redundantResults {
		if len(redundantResult) == 0 {
			resultsByID[resultID].ScanResultData.Redundancy = fixLabel
		} else {
			resultsByID[resultID].ScanResultData.Redundancy = redundantLabel
		}
	}
}

func computeRedundantResults(subFlows map[string]*SubFlow, queryResults []*wrappers.ScanResult) map[string]map[string]bool {
	redundantResults := make(map[string]map[string]bool)
	for _, result := range queryResults {
		redundantResults[result.ID] = make(map[string]bool)
	}

	sortedSubFlowIDs := sortSubFlowIDs(subFlows)
	for _, key := range sortedSubFlowIDs {
		sortedResults := sortSubFlowResultIDs(subFlows[key])
		coverage := make(map[string]float64)
		for _, resultID := range sortedResults {
			result := getResultForID(queryResults, resultID)
			coverage[resultID] = roundFloat(float64(len(subFlows[key].Flow))/float64(len(result.ScanResultData.Nodes)), precision)
		}
		maxCoverageResultID, maxCoverage := getMaxCoverage(coverage)

		if maxCoverage < half {
			continue
		}
		for resultID := range coverage {
			if resultID == maxCoverageResultID {
				continue
			}
			redundantResults[resultID][maxCoverageResultID] = true
		}
	}
	return redundantResults
}

func getMaxCoverage(coverage map[string]float64) (maxCoverageResultID string, maxCoverage float64) {
	var sortedResultsIDs []string
	for resultID := range coverage {
		sortedResultsIDs = append(sortedResultsIDs, resultID)
	}
	sort.Strings(sortedResultsIDs)

	maxCoverageResultID = ""
	maxCoverage = 0.0
	for _, resultID := range sortedResultsIDs {
		c := coverage[resultID]
		if c > maxCoverage {
			maxCoverage = c
			maxCoverageResultID = resultID
		}
	}
	return maxCoverageResultID, maxCoverage
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(ten, float64(precision))
	return math.Round(val*ratio) / ratio
}

func getResultForID(queryResults []*wrappers.ScanResult, resultID string) *wrappers.ScanResult {
	for _, result := range queryResults {
		if result.ID == resultID {
			return result
		}
	}
	return nil
}

func sortSubFlowIDs(subFlows map[string]*SubFlow) []string {
	var subFlowIDs []string
	for key := range subFlows {
		subFlowIDs = append(subFlowIDs, key)
	}
	sort.Strings(subFlowIDs)
	return subFlowIDs
}

func sortSubFlowResultIDs(subFlow *SubFlow) []string {
	var results []string
	for result := range subFlow.Results {
		results = append(results, result)
	}
	sort.Strings(results)
	return results
}

func buildFlows(queryResults []*wrappers.ScanResult) map[string][]string {
	flowsByResult := make(map[string][]string)
	for _, result := range queryResults {
		var resultNodes []string
		for _, node := range result.ScanResultData.Nodes {
			nodeStr := fmt.Sprintf("%s:%d:%d", node.FileName, node.Line, node.Column)
			resultNodes = append(resultNodes, nodeStr)
		}
		flowsByResult[result.ID] = resultNodes
	}
	return flowsByResult
}

type SubFlow struct {
	ShaOne  string
	Flow    []string
	Results map[string]bool
}

func compareFlows(flows map[string][]string) map[string]*SubFlow {
	subFlows := make(map[string]*SubFlow)
	comparedFlows := make(map[string]bool)
	for r1, f1 := range flows {
		for r2, f2 := range flows {
			if r1 == r2 {
				continue
			}
			key := GetKey(r1, r2)
			if _, exist := comparedFlows[key]; exist {
				continue
			}
			comparedFlows[key] = true

			// compute the subflow of f1 and f2
			exist, sf := computeSubFlow(f1, f2)
			if !exist {
				continue
			}
			if _, exist := subFlows[sf.ShaOne]; !exist {
				sf.Results = make(map[string]bool)
				subFlows[sf.ShaOne] = sf
			}
			subFlows[sf.ShaOne].Results[r1] = true
			subFlows[sf.ShaOne].Results[r2] = true
		}
	}
	return subFlows
}

func GetKey(r1, r2 string) string {
	if r1 <= r2 {
		return r1 + "," + r2
	}
	return r2 + "," + r1
}

func computeSubFlow(f1, f2 []string) (bool, *SubFlow) {
	var subFlow []string
	for i1 := 0; i1 < len(f1); {
		for i2 := 0; i2 < len(f2) && i1 < len(f1); {
			if f1[i1] == f2[i2] {
				subFlow = append(subFlow, f1[i1])
				i1++
				i2++
			} else {
				if len(subFlow) > 0 {
					break
				}
				i2++
			}
		}
		if len(subFlow) > 0 {
			break
		}
		i1++
	}
	if len(subFlow) == 0 {
		return false, nil
	}
	sha1String := getSha1String(subFlow)
	return true, &SubFlow{sha1String, subFlow, nil}
}

func getSha1String(lines []string) string {
	h := sha1.New()

	// Write the bytes of the input string into the hash
	h.Write([]byte(strings.Join(lines, "")))

	// Get the final hash result as a byte slice
	bs := h.Sum(nil)

	// Convert the byte slice to a hexadecimal string
	sha1String := hex.EncodeToString(bs)

	return sha1String
}
