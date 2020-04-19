package wrappers

type ResultsWrapper interface {
	GetByScanID(scanID string, limit, offset uint64) (*ResultsResponseModel, *ResultError, error)
}

type ResultError struct {
	Code    int32       `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Result is based on results.ResultRow
type ResultResponseModel struct {
	// query
	QueryID   int64  `json:"queryID,omitempty"`
	QueryName string `json:"queryName,omitempty"`
	Severity  string `json:"severity,omitempty"`
	CweID     int64  `json:"cweID,omitempty"`
	// path
	SimilarityID    int64         `json:"similarityID,omitempty"`
	UniqueID        int64         `json:"uniqueID,omitempty"`
	Nodes           []*ResultNode `json:"nodes,omitempty"`
	ConfidenceLevel float32       `json:"confidenceLevel,omitempty"`
	// query
	Groups []string `json:"groups,omitempty"`
	// path
	PathSystemID                    string `json:"pathSystemID,omitempty"`
	PathSystemIDBySimiAndFilesPaths string `json:"pathSystemIDBySimiAndFilesPaths,omitempty"`

	ID           string `json:"id,omitempty"`
	FirstScanID  string `json:"firstScanID,omitempty"`
	FirstFoundAt string `json:"firstFoundAt,omitempty"`
	FoundAt      string `json:"foundAt,omitempty"`
	Status       string `json:"status,omitempty"`
}

type ResultNode struct {
	Column       int32  `json:"column,omitempty"`
	FileName     string `json:"fileName,omitempty"`
	FullName     string `json:"fullName,omitempty"`
	Length       int32  `json:"length,omitempty"`
	Line         int32  `json:"line,omitempty"`
	MethodLine   int32  `json:"methodLine,omitempty"`
	Name         string `json:"name,omitempty"`
	NodeID       int32  `json:"-"`
	DomType      string `json:"domType,omitempty"`
	NodeSystemID string `json:"nodeSystemID,omitempty"`
}

type ResultsResponseModel struct {
	Results    []ResultResponseModel
	TotalCount int
}
