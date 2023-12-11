package sastchat

import (
	"encoding/json"
	"fmt"
	"os"
)

// Define the Go structs that match the JSON structure
type ResultPartial struct {
	Type                 string                 `json:"type"`
	Label                string                 `json:"label"`
	ID                   string                 `json:"id"`
	SimilarityID         string                 `json:"similarityId"`
	Status               string                 `json:"status"`
	State                string                 `json:"state"`
	Severity             string                 `json:"severity"`
	Created              string                 `json:"created"`
	FirstFoundAt         string                 `json:"firstFoundAt"`
	FoundAt              string                 `json:"foundAt"`
	FirstScanID          string                 `json:"firstScanId"`
	Description          string                 `json:"description"`
	DescriptionHTML      string                 `json:"descriptionHTML"`
	Data                 json.RawMessage        `json:"data"`
	Comments             map[string]interface{} `json:"comments"`
	VulnerabilityDetails json.RawMessage        `json:"vulnerabilityDetails"`
}
type Result struct {
	Type                 string                 `json:"type"`
	Label                string                 `json:"label"`
	ID                   string                 `json:"id"`
	SimilarityID         string                 `json:"similarityId"`
	Status               string                 `json:"status"`
	State                string                 `json:"state"`
	Severity             string                 `json:"severity"`
	Created              string                 `json:"created"`
	FirstFoundAt         string                 `json:"firstFoundAt"`
	FoundAt              string                 `json:"foundAt"`
	FirstScanID          string                 `json:"firstScanId"`
	Description          string                 `json:"description"`
	DescriptionHTML      string                 `json:"descriptionHTML"`
	Data                 Data                   `json:"data"`
	Comments             map[string]interface{} `json:"comments"`
	VulnerabilityDetails VulnerabilityDetails   `json:"vulnerabilityDetails"`
}

type Data struct {
	QueryID      uint64 `json:"queryId"`
	QueryName    string `json:"queryName"`
	Group        string `json:"group"`
	ResultHash   string `json:"resultHash"`
	LanguageName string `json:"languageName"`
	Nodes        []Node `json:"nodes"`
}

type Node struct {
	ID          string `json:"id"`
	Line        int    `json:"line"`
	Name        string `json:"name"`
	Column      int    `json:"column"`
	Length      int    `json:"length"`
	Method      string `json:"method"`
	NodeID      int    `json:"nodeID"`
	DomType     string `json:"domType"`
	FileName    string `json:"fileName"`
	FullName    string `json:"fullName"`
	TypeName    string `json:"typeName"`
	MethodLine  int    `json:"methodLine"`
	Definitions string `json:"definitions"`
}

type VulnerabilityDetails struct {
	CweID       int                    `json:"cweId"`
	Cvss        map[string]interface{} `json:"cvss"`
	Compliances []string               `json:"compliances"`
}

type ScanResultsPartial struct {
	Results    json.RawMessage `json:"results"`
	TotalCount int             `json:"totalCount"`
	ScanID     string          `json:"scanID"`
}

type ScanResults struct {
	Results    []Result `json:"results"`
	TotalCount int      `json:"totalCount"`
	ScanID     string   `json:"scanID"`
}

func ReadResultsAll(filename string) (*ScanResults, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON data into the ScanResults struct
	var scanResults ScanResults
	err = json.Unmarshal(bytes, &scanResults)
	if err != nil {
		return nil, err
	}

	return &scanResults, nil
}

func ReadResultsSAST(filename string) (*ScanResults, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON data into the ScanResults struct
	var scanResultsPartial ScanResultsPartial
	if err = json.Unmarshal(bytes, &scanResultsPartial); err != nil {
		return nil, err
	}

	var results []Result
	var resultsPartial []ResultPartial
	if err = json.Unmarshal(scanResultsPartial.Results, &resultsPartial); err != nil {
		return nil, err
	}

	for _, resultPartial := range resultsPartial {
		if resultPartial.Type == "sast" {
			var data Data
			if err = json.Unmarshal(resultPartial.Data, &data); err != nil {
				return nil, err
			}
			var vulnerabilityDetails VulnerabilityDetails
			if err = json.Unmarshal(resultPartial.VulnerabilityDetails, &vulnerabilityDetails); err != nil {
				return nil, err
			}

			result := Result{resultPartial.Type,
				resultPartial.Label,
				resultPartial.ID,
				resultPartial.SimilarityID,
				resultPartial.Status,
				resultPartial.State,
				resultPartial.Severity,
				resultPartial.Created,
				resultPartial.FirstFoundAt,
				resultPartial.FoundAt,
				resultPartial.FirstScanID,
				resultPartial.Description,
				resultPartial.DescriptionHTML,
				data,
				resultPartial.Comments,
				vulnerabilityDetails}
			results = append(results, result)
		}
	}
	scanResults := ScanResults{results, scanResultsPartial.TotalCount, scanResultsPartial.ScanID}
	return &scanResults, nil
}

func GetLanguages(results *ScanResults, language string) map[string]bool {
	languages := make(map[string]bool)
	if language == "" {
		for _, result := range results.Results {
			languages[result.Data.LanguageName] = true
		}
	} else {
		languages[language] = true
	}
	return languages
}

func GetQueries(results *ScanResults, languages map[string]bool, query string) map[string]map[string]bool {
	queriesByLanguage := make(map[string]map[string]bool)
	if query == "" {
		for _, result := range results.Results {
			if _, exist := languages[result.Data.LanguageName]; !exist {
				continue
			}
			if _, exist := queriesByLanguage[result.Data.LanguageName]; !exist {
				queriesByLanguage[result.Data.LanguageName] = make(map[string]bool)
			}
			queriesByLanguage[result.Data.LanguageName][result.Data.QueryName] = true
		}
		//fmt.Printf("There are %d Java queries to process\n", len(queriesByLanguage["Java"]))
	} else {
		for language := range languages {
			queriesByLanguage[language] = make(map[string]bool)
			queriesByLanguage[language][query] = true
		}
	}
	return queriesByLanguage
}

func GetResultById(results *ScanResults, resultId string) (Result, error) {
	for _, result := range results.Results {
		if result.ID == resultId {
			return result, nil
		}
	}
	return Result{}, fmt.Errorf("result ID %s not found", resultId)
}
