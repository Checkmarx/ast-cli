package wrappers

// ScanResultsCollection
// NOTE: This should be read from scan-results library but that
// isn't compatible with the mocked data from the results API???
type ScanResultsCollection struct {
	Results    []*ScanResult `json:"results"`
	TotalCount uint          `json:"totalCount"`
}

type ScanResult struct {
	VulnerabilityDetails
	ScanResultData
	Type            string `json:"type,omitempty"`
	ID              string `json:"id,omitempty"`
	SimilarityID    string `json:"similarityID,omitempty"`
	Status          string `json:"status,omitempty"`
	State           string `json:"state,omitempty"`
	Severity        string `json:"severity,omitempty"`
	FirstFoundAt    string `json:"firstFoundAt,omitempty"`
	FoundAt         string `json:"foundAt,omitempty"`
	FirstScan       string `json:"firstScan,omitempty"`
	ConfidenceLevel int32  `json:"confidenceLevel,omitempty"`
	Group           string `json:"group,omitempty"`
	ResultHash      string `json:"resultHash,omitempty"`
	LanguageName    string `json:"languageName,omitempty"`
	FirstScanID     string `json:"firstScanID,omitempty"`
	UpdateAt        string `json:"updateAt,omitempty"`
}

type VulnerabilityDetails struct {
	CveName            string `json:"cveName,omitempty"`
	CVSS               string `json:"cvss*,omitempty"`
	CvssScore          string `json:"cvssScore,omitempty"`
	CweID              string `json:"cweId,omitempty"`
	Owasp2017          string `json:"owasp2017,omitempty"`
	CopyLeft           string `json:"copyLeft,omitempty"`
	CopyrightRiskScore string `json:"copyrightRiskScore,omitempty"`
	Linking            string `json:"linking,omitempty"`
	PatentRiskScore    string `json:"patentRiskScore,omitempty"`
	RoyaltyFree        string `json:"royaltyFree,omitempty"`
}

type ScanResultNode struct {
	Column     string `json:"column,omitempty"`
	FileName   string `json:"fileName,omitempty"`
	FullName   string `json:"fullName,omitempty"`
	Length     string `json:"length,omitempty"`
	Line       string `json:"line,omitempty"`
	MethodLine string `json:"methodLine,omitempty"`
	DomType    string `json:"domType,omitempty"`
	NodeHash   string `json:"nodeHash,omitempty"`
}

type ScanResultPackageData struct {
	Comment string `json:"comment,omitempty"`
	Type    string `json:"type,omitempty"`
	URL     string `json:"url,omitempty"`
}

type ScanResultExploitableMethods struct {
	Exploit string `json:"exploit,omitempty"`
}

type ScanResultData struct {
	QueryID            string `json:"queryID,omitempty"`
	QueryName          string `json:"queryName,omitempty"`
	Severity           string `json:"severity,omitempty"`
	CweID              string `json:"cweID,omitempty"`
	SimilarityID       string `json:"similarityID,omitempty"`
	UniqueID           string `json:"uniqueID,omitempty"`
	Description        string `json:"description,omitempty"`
	Recommendation     string `json:"recommendation,omitempty"`
	PackageID          string `json:"packageId,omitempty"`
	RecommendedVersion string `json:"recommendedVersion,omitempty"`
	PackagePublishAt   string `json:"packagePublishAt,omitempty"`
	QueryURL           string `json:"queryURL,omitempty"`
	PackageType        string `json:"packageType,omitempty"`
	PackageURL         string `json:"packageURL,omitempty"`
	Group              string `json:"group,omitempty"`
	FileName           string `json:"fileName,omitempty"`
	Line               string `json:"line,omitempty"`
	Platform           string `json:"platform,omitempty"`
	IssueType          string `json:"issueType,omitempty"`
	SearchKey          string `json:"searchKey,omitempty"`
	ExpectedValue      string `json:"expectedValue,omitempty"`
	Value              string `json:"value,omitempty"`
	// Single object
	ScanResultExploitableMethods
	// This is an array
	ScanResultNode
	// This an array
	ScanResultPackageData
}
