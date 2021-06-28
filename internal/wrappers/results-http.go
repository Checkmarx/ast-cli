package wrappers

import (
	"encoding/json"
	"net/http"

	"github.com/checkmarxDev/sast-results/pkg/reader"
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	resultsRaw "github.com/checkmarxDev/sast-results/pkg/web/path/raw"

	"github.com/pkg/errors"
)

const (
	failedToParseGetResults = "Failed to parse list results"
)

type ResultsHTTPWrapper struct {
	path     string
	sastPath string
	kicsPath string
}

func NewHTTPResultsWrapper(path string, sastPath string, kicsPath string) ResultsWrapper {
	return &ResultsHTTPWrapper{
		path:     path,
		sastPath: sastPath,
		kicsPath: kicsPath,
	}
}

func (r *ResultsHTTPWrapper) GetSastByScanID(params map[string]string) (*resultsRaw.ResultsCollection, *resultsHelpers.WebError, error) {
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.sastPath, params, nil, DefaultTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := resultsHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := resultsRaw.ResultsCollection{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r *ResultsHTTPWrapper) GetKicsByScanID(params map[string]string) (*resultsRaw.ResultsCollection, *resultsHelpers.WebError, error) {
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.kicsPath, params, nil, DefaultTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := resultsHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := KicsResultsCollection{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return normalizeKicsResult(&model), nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func normalizeKicsResult(srcResult *KicsResultsCollection) *resultsRaw.ResultsCollection {
	model := resultsRaw.ResultsCollection{}
	for idx := range srcResult.Results {
		result := srcResult.Results[idx]
		r := reader.Result{}
		// RESULT NODE IS MISSING Type. ALSO type seems to mean something different on the kics value
		//r.Type = "kics"
		r.ID = result.ID
		// Can't map QueryID to correct type
		//r.ResultQuery.QueryID = srcResult.Results[0].QueryID
		r.ResultQuery.QueryName = result.QueryName
		r.ResultQuery.Severity = result.Severity
		// String to int64?
		//r.SimilarityID = srcResult.Results[0].SimilarityID
		r.Group = result.Group
		r.FirstScanID = result.FirstScanID
		r.FirstFoundAt = result.FirstFoundAt
		r.Status = result.Status
		r.FoundAt = result.FoundAt
		r.State = result.State
		// Create the result node
		rNode := reader.ResultNode{}
		rNode.FileName = result.FileName
		rNode.Line = result.Line
		r.Nodes = append(r.Nodes, &rNode)
		model.Results = append(model.Results, &r)
	}
	model.TotalCount = srcResult.TotalCount
	return &model
}

/*
 * NOTE: these Kics data structures a ripped of mods from whats in
 * what's in sast-results/results.go. This shim code until we can figure out
 * how to bring these items together. IT DOES NOT BELONG HERE.
 */
type KicsResultsCollection struct {
	Results    []*KicsResult `json:"results"`
	TotalCount uint          `json:"totalCount"`
}

//"ID":"SvWMXc05xEEs3mn3kwLwASgchd8=",
//"similarityID":"cc02172daa86dc2804201e98627f455246dbb7284a58a1a3d491135e3a4aee8a",
//"severity":"HIGH",
//"firstScanID":"adfc597a-825e-4d1f-ba4e-07eb556b4fc7",
//"firstFoundAt":"0001-01-01T00:00:00Z",
//"foundAt":"0001-01-01T00:00:00Z",
//"status":"NEW",
//"state":"TO_VERIFY",
//"type":"Missing Attribute",
//"queryID":"fd54f200-402c-4333-a5a4-36ef6709af2f",
//"queryName":"Missing User Instruction",
//"group":"fd54f200-402c-4333-a5a4-36ef6709af2f",
//"queryURL":"Missing User Instruction",
//"fileName":"/dvna-master/Dockerfile",
//"line":3,
//"platform":"Dockerfile",
//"issueType":"Missing Attribute",
//"searchKey":"Missing Attribute",
//"searchValue":"3",
//"expectedValue":"The 'Dockerfile' contains the 'USER' instruction",
//"actualValue":"The 'Dockerfile' does not contain any 'USER' instruction",
//"value":"The 'Dockerfile' does not contain any 'USER' instruction",
//"description":"/dvna-master/Dockerfile",
//"comments":"/dvna-master/Dockerfile",
//"category":"Build Process"},

type KicsResult struct {
	// Was ResultQuery structure
	// Should this be a string? it was uint64
	QueryID   string `json:"queryID,omitempty"`
	QueryName string `json:"queryName,omitempty"`
	Severity  string `json:"severity,omitempty"`
	//CweID     int64  `json:"cweID,omitempty"`
	// I swapped SimilarityID from int64 to string
	//SimilarityID    int64         `json:"similarityID,omitempty"`
	SimilarityID string `json:"similarityID,omitempty"`
	// UniqueID doesn't exist on kics
	//UniqueID        int64         `json:"uniqueID,omitempty"`
	// Nodes does not exist on kics
	//Nodes           []*ResultNode `json:"nodes,omitempty"`
	// Confidence level does not exist on kics
	//ConfidenceLevel float32       `json:"confidenceLevel,omitempty"`
	// query
	Group string `json:"group,omitempty"`
	// PathSystemID does not exist on kics
	//PathSystemID string `json:"pathSystemID,omitempty"`
	// LanguageName does not exist on kics
	//LanguageName string `json:"languageName,omitempty"`
	ID          string `json:"id,omitempty"`
	FirstScanID string `json:"firstScanID,omitempty"`
	// Should this be a date data structure?
	FirstFoundAt string `json:"firstFoundAt,omitempty"`
	// Should this be a date data structure?
	FoundAt string `json:"foundAt,omitempty"`
	Status  string `json:"status,omitempty"`
	// Type did not exist, also in sample it was sast, sca, etc -- its different here
	Type string `json:"type,omitempty"`
	// QueryURL did not exist
	QueryURL string `json:"queryURL,omitempty"`
	State    string `json:"state,omitempty"`
	// Platform didn't exist in SAST
	Platform string `json:"platform,omitempty"`
	//
	/// start: None of the following values exist in SAST
	//
	IssueType     string `json:"issueType,omitempty"`
	SearchKey     string `json:"searchKey,omitempty"`
	SearchValue   string `json:"searchValue,omitempty"`
	ExpectedValue string `json:"expectedValue,omitempty"`
	ActualValue   string `json:"actualValue,omitempty"`
	Value         string `json:"value,omitempty"`
	Description   string `json:"description,omitempty"`
	Comments      string `json:"comments,omitempty"`
	Category      string `json:"category,omitempty"`
	//
	/// End: None of the following values exist in SAST
	//
	// ProjectId does not exist in kics
	//ProjectID string `json:"-"` // not exported
	// TenantID does not exist in kics
	//TenantID  string `json:"-"` // not exported
	//
	/// These were found in ResultNode in Sast structure
	//
	// Column does not exist in kics
	//Column       int32  `json:"column,omitempty"`
	FileName string `json:"fileName,omitempty"`
	// Fullname does not exist in kics
	//FullName string `json:"fullName,omitempty"`
	// Lengh does not exist in kics
	//Length       int32  `json:"length,omitempty"`
	Line int32 `json:"line,omitempty"`
	// MethodLine does not exist in kics
	//MethodLine   int32  `json:"methodLine,omitempty"`
	// Name does not make sense in kics
	// Name         string `json:"name,omitempty"`
	// NodeID does not exist in kics
	//NodeID       int32  `json:"-"`
	// DomType does not exist in kics
	// DomType      string `json:"domType,omitempty"`
	// NodeSystemID does not exist in kics
	//NodeSystemID string `json:"nodeSystemID,omitempty"` ,
}
