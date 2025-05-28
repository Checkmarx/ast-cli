package wrappers

type RealtimeScannerWrapper interface {
	Scan(packages *RealtimeScannerPackageRequest) (*RealtimeScannerPackageResponse, error)
}

type RealtimeScannerResults struct {
	PackageManager  string                         `json:"PackageManager"`
	PackageName     string                         `json:"PackageName"`
	Version         string                         `json:"PackageVersion"`
	Status          string                         `json:"Status"`
	Vulnerabilities []RealtimeScannerVulnerability `json:"Vulnerabilities"`
}

type RealtimeScannerVulnerability struct {
	CVE         string `json:"CVE"`
	Description string `json:"Description"`
	Severity    string `json:"Severity"`
}

type RealtimeScannerPackageResponse struct {
	Packages []RealtimeScannerResults `json:"Packages"`
}

type RealtimeScannerPackage struct {
	PackageManager string `json:"PackageManager"`
	PackageName    string `json:"PackageName"`
	Version        string `json:"PackageVersion"`
}

type RealtimeScannerPackageRequest struct {
	Packages []RealtimeScannerPackage `json:"Packages"`
}
