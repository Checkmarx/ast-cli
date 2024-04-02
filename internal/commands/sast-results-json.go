package commands

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
	Results    []*Result `json:"results"`
	TotalCount int       `json:"totalCount"`
	ScanID     string    `json:"scanID"`
}

func ReadResultsSAST(filename string) (*ScanResults, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON data into the ScanResults struct
	var scanResultsPartial ScanResultsPartial
	if err := json.Unmarshal(bytes, &scanResultsPartial); err != nil {
		return nil, err
	}

	var results []*Result
	var resultsPartial []*ResultPartial
	if err := json.Unmarshal(scanResultsPartial.Results, &resultsPartial); err != nil {
		return nil, err
	}

	for _, resultPartial := range resultsPartial {
		if resultPartial.Type != "sast" {
			continue
		}
		var data Data
		if err := json.Unmarshal(resultPartial.Data, &data); err != nil {
			return nil, err
		}
		var vulnerabilityDetails VulnerabilityDetails
		if err := json.Unmarshal(resultPartial.VulnerabilityDetails, &vulnerabilityDetails); err != nil {
			return nil, err
		}

		result := &Result{resultPartial.Type,
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
	scanResults := ScanResults{results, scanResultsPartial.TotalCount, scanResultsPartial.ScanID}
	return &scanResults, nil
}

func GetResultByID(results *ScanResults, resultID string) (*Result, error) {
	for _, result := range results.Results {
		if result.ID == resultID {
			return result, nil
		}
	}
	return &Result{}, fmt.Errorf("result ID %s not found", resultID)
}
