package ossrealtime

// OssPackage represents a package's details for OSS scanning.
type OssPackage struct {
	PackageManager  string          `json:"PackageManager"`
	PackageName     string          `json:"PackageName"`
	PackageVersion  string          `json:"PackageVersion"`
	FilePath        string          `json:"FilePath"`
	LineStart       int             `json:"LineStart"`
	LineEnd         int             `json:"LineEnd"`
	StartIndex      int             `json:"StartIndex"`
	EndIndex        int             `json:"EndIndex"`
	Status          string          `json:"Status"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities"`
}

// OssPackageResults holds the results of an OSS scan.
type OssPackageResults struct {
	Packages []OssPackage `json:"Packages"`
}

type Vulnerability struct {
	CVE         string `json:"CVE"`
	Description string `json:"Description"`
	Severity    string `json:"Severity"`
}
