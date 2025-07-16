package wrappers

type RealtimeScannerWrapper interface {
	ScanPackages(packages *RealtimeScannerPackageRequest) (*RealtimeScannerPackageResponse, error)
	ScanImages(images *ContainerImageRequest) (*ContainerImageResponse, error)
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

type ContainerImageRequest struct {
	Images []ContainerImageRequestItem `json:"images"`
}

type ContainerImageRequestItem struct {
	ImageName string `json:"imageName"`
	ImageTag  string `json:"imageTag"`
}

type ContainerImageResponse struct {
	Images []ContainerImageResponseItem `json:"images"`
}

type ContainerImageResponseItem struct {
	ImageName       string                        `json:"imageName"`
	ImageTag        string                        `json:"imageTag"`
	Status          string                        `json:"status"`
	Vulnerabilities []ContainerImageVulnerability `json:"vulnerabilities"`
}

type ContainerImageVulnerability struct {
	CVE         string `json:"CVE"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}
