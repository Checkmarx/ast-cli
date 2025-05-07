package wrappers

type OssRealtimeWrapper interface {
	Scan(packages []OssPackageRequest) (OssResults, error)
}

type OssPackage struct {
	PackageManager string `json:"PackageManager"`
	PackageName    string `json:"PackageName"`
	Version        string `json:"Version"`
	Status         string `json:"Status,omitempty"`
}

type OssResults struct {
	Packages []OssPackage `json:"Packages"`
}

type OssPackageRequest struct {
	PackageManager string `json:"PackageManager"`
	PackageName    string `json:"PackageName"`
	Version        string `json:"Version"`
}
