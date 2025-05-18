package wrappers

type RealtimeScannerWrapper interface {
	Scan(packages *OssPackageRequest) (*OssPackageResponse, error)
}

type OssResults struct {
	PackageManager string `json:"PackageManager"`
	PackageName    string `json:"PackageName"`
	Version        string `json:"PackageVersion"`
	Status         string `json:"Status,omitempty"`
}

type OssPackageResponse struct {
	Packages []OssResults `json:"Packages"`
}

type OssPackage struct {
	PackageManager string `json:"PackageManager"`
	PackageName    string `json:"PackageName"`
	Version        string `json:"PackageVersion"`
}

type OssPackageRequest struct {
	Packages []OssPackage `json:"Packages"`
}
