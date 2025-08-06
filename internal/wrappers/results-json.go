package wrappers

type ScaRealtimeScanResultsCollection struct {
	Results    []ScanResult           `json:"results"`
	Errors     []ScaRealtimeScanError `json:"errors"`
	TotalCount uint                   `json:"totalCount"`
	ScanID     string                 `json:"scanID"`
}

type ScaRealtimeScanError struct {
	Filename string `json:"filename"`
	Message  string `json:"message"`
}

type ScanResultsCollection struct {
	Results    []*ScanResult `json:"results"`
	TotalCount uint          `json:"totalCount"`
	ScanID     string        `json:"scanID"`
}

type ScanResult struct {
	Type                 string               `json:"type,omitempty"`
	ScaType              string               `json:"scaType,omitempty"`
	Label                string               `json:"label,omitempty"`
	ID                   string               `json:"id,omitempty"`
	SimilarityID         string               `json:"similarityId,omitempty"`
	AlternateID          string               `json:"alternateId,omitempty"`
	Status               string               `json:"status,omitempty"`
	State                string               `json:"state,omitempty"`
	Severity             string               `json:"severity,omitempty"`
	Created              string               `json:"created,omitempty"`
	FirstFoundAt         string               `json:"firstFoundAt,omitempty"`
	FoundAt              string               `json:"foundAt,omitempty"`
	FirstScan            string               `json:"firstScan,omitempty"`
	FirstScanID          string               `json:"firstScanId,omitempty"`
	PublishedAt          string               `json:"publishedAt,omitempty"`
	Recommendations      string               `json:"recommendations,omitempty"`
	Description          string               `json:"description,omitempty"`
	DescriptionHTML      string               `json:"descriptionHTML,omitempty"`
	ScanResultData       ScanResultData       `json:"data,omitempty"`
	Comments             ResultComments       `json:"comments,omitempty"`
	VulnerabilityDetails VulnerabilityDetails `json:"vulnerabilityDetails,omitempty"`
}

type ResultComments struct {
	Comments string `json:"comments,omitempty"`
}

type VulnerabilityDetails struct {
	CweID       interface{}       `json:"cweId,omitempty"`
	CvssScore   float64           `json:"cvssScore,omitempty"`
	CveName     string            `json:"cveName,omitempty"`
	CVSS        VulnerabilityCVSS `json:"cvss,omitempty"`
	Compliances []*string         `json:"compliances,omitempty"`
}

type VulnerabilityCVSS struct {
	Version            int    `json:"version,omitempty"`
	AttackVector       string `json:"attackVector,omitempty"`
	Availability       string `json:"availability,omitempty"`
	Confidentiality    string `json:"confidentiality,omitempty"`
	AttackComplexity   string `json:"attackComplexity,omitempty"`
	IntegrityImpact    string `json:"integrityImpact,omitempty"`
	Scope              string `json:"scope,omitempty"`
	PrivilegesRequired string `json:"privilegesRequired,omitempty"`
	UserInteraction    string `json:"userInteraction,omitempty"`
}

type ScanResultNode struct {
	ID          string `json:"id,omitempty"`
	Line        uint   `json:"line"`
	Name        string `json:"name,omitempty"`
	Column      uint   `json:"column"`
	Length      uint   `json:"length,omitempty"`
	Method      string `json:"method,omitempty"`
	NodeID      int    `json:"nodeID,omitempty"`
	DomType     string `json:"domType,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	FullName    string `json:"fullName,omitempty"`
	TypeName    string `json:"typeName,omitempty"`
	MethodLine  uint   `json:"methodLine,omitempty"`
	Definitions string `json:"definitions,omitempty"`
}

type ScanResultPackageData struct {
	Comment string `json:"comment,omitempty"`
	Type    string `json:"type,omitempty"`
	URL     string `json:"url,omitempty"`
}

type ScanResultData struct {
	QueryID              interface{}              `json:"queryId,omitempty"`
	QueryName            string                   `json:"queryName,omitempty"`
	Group                string                   `json:"group,omitempty"`
	ResultHash           string                   `json:"resultHash,omitempty"`
	LanguageName         string                   `json:"languageName,omitempty"`
	Redundancy           string                   `json:"redundancy,omitempty"`
	Description          string                   `json:"description,omitempty"`
	Nodes                []*ScanResultNode        `json:"nodes,omitempty"`
	PackageData          []*ScanResultPackageData `json:"packageData,omitempty"`
	PackageID            []*ScanResultPackageData `json:"packageId,omitempty"`
	PackageIdentifier    string                   `json:"packageIdentifier,omitempty"`
	PublishedAt          string                   `json:"publishedAt,omitempty"`
	ScaPackageCollection *ScaPackageCollection    `json:"scaPackageData,omitempty"`
	RecommendedVersion   interface{}              `json:"recommendedVersion,omitempty"`
	// Added to support kics results
	Line          uint   `json:"line,omitempty"` // also used by SSCS results
	Platform      string `json:"platform,omitempty"`
	IssueType     string `json:"issueType,omitempty"`
	ExpectedValue string `json:"expectedValue,omitempty"`
	Value         string `json:"value,omitempty"`
	Filename      string `json:"filename,omitempty"` // also used by SSCS results
	// Added to support containers results
	PackageName    string `json:"packageName,omitempty"`
	PackageVersion string `json:"packageVersion,omitempty"`
	ImageName      string `json:"imageName,omitempty"`
	ImageTag       string `json:"imageTag,omitempty"`
	ImageFilePath  string `json:"imageFilePath,omitempty"`
	ImageOrigin    string `json:"imageOrigin,omitempty"`
	// Added to support SSCS results
	RuleID                *string `json:"ruleId,omitempty"`
	RuleName              string  `json:"ruleName,omitempty"`
	Snippet               string  `json:"snippet,omitempty"`
	SlsaStep              string  `json:"slsaStep,omitempty"`
	RuleDescription       string  `json:"ruleDescription,omitempty"`
	Remediation           string  `json:"remediation,omitempty"`
	RemediationLink       string  `json:"remediationLink,omitempty"`
	RemediationAdditional string  `json:"remediationAdditional,omitempty"`
	Validity              string  `json:"validity,omitempty"`
}
