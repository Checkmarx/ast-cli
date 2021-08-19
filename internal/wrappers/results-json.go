package wrappers

// ScanResultsCollection
// NOTE: This should be read from scan-results library but that
// isn't compatible with the mocked data from the results API???
type ScanResultsCollection struct {
	Results    []*ScanResult `json:"results"`
	TotalCount uint          `json:"totalCount"`
}

type ScanResult struct {
	Type                 string               `json:"type,omitempty"`
	ID                   string               `json:"id,omitempty"`
	SimilarityID         string               `json:"similarityID,omitempty"`
	Status               string               `json:"status,omitempty"`
	State                string               `json:"state,omitempty"`
	Severity             string               `json:"severity,omitempty"`
	FirstFoundAt         string               `json:"firstFoundAt,omitempty"`
	FoundAt              string               `json:"foundAt,omitempty"`
	FirstScan            string               `json:"firstScan,omitempty"`
	FirstScanID          string               `json:"firstScanID,omitempty"`
	PublishedAt          string               `json:"publishedAt,omitempty"`
	Created              string               `json:"created,omitempty"`
	Recommendations      string               `json:"recommendations,omitempty"`
	Comments             ResultComments       `json:"Comments,omitempty"`
	VulnerabilityDetails VulnerabilityDetails `json:"vulnerabilityDetails,omitempty"`
	ScanResultData       ScanResultData       `json:"data,omitempty"`
}

type ResultComments struct {
	Comments string `json:"comments,omitempty"`
}

type VulnerabilityDetails struct {
	CvssScore   float64           `json:"cvssScore,omitempty"`
	CveName     string            `json:"cveName,omitempty"`
	CVSS        VulnerabilityCVSS `json:"cvss,omitempty"`
	Compliances []*string         `json:"compliances,omitempty"`
	// CweID       string            `json:"cweId,string"`
}

type VulnerabilityCVSS struct {
	Version          int    `json:"version,omitempty"`
	AttackVector     string `json:"attackVector,omitempty"`
	Availability     string `json:"availability,omitempty"`
	Confidentiality  string `json:"confidentiality,omitempty"`
	AttackComplexity string `json:"attackComplexity,omitempty"`
}

type ScanResultNode struct {
	ID         string `json:"id,omitempty"`
	Line       int    `json:"line,omitempty"`
	Name       string `json:"name,omitempty"`
	Column     int    `json:"column,omitempty"`
	Length     int    `json:"length,omitempty"`
	NodeID     int    `json:"nodeID,omitempty"`
	DomType    string `json:"domType,omitempty"`
	FileName   string `json:"fileName,omitempty"`
	FullName   string `json:"fullName,omitempty"`
	MethodLine int    `json:"methodLine,omitempty"`
}

type ScanResultPackageData struct {
	Comment string `json:"comment,omitempty"`
	Type    string `json:"type,omitempty"`
	URL     string `json:"url,omitempty"`
}

type ScanResultData struct {
	QueryID      int                      `json:"queryIDFoo,omitempty"`
	QueryName    string                   `json:"queryName,omitempty"`
	Group        string                   `json:"group,omitempty"`
	ResultHash   string                   `json:"resultHash,omitempty"`
	LanguageName string                   `json:"languageName,omitempty"`
	Description  string                   `json:"description,omitempty"`
	Nodes        []*ScanResultNode        `json:"nodes"`
	PackageData  []*ScanResultPackageData `json:"packageData,omitempty"`
	PackageID    []*ScanResultPackageData `json:"packageId,omitempty"`
}
