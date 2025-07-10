package containersrealtime

import "github.com/checkmarx/ast-cli/internal/services/realtimeengine"

// ContainerImage represents a container image's details for realtime scanning.
type ContainerImage struct {
	ImageName       string                    `json:"ImageName"`
	ImageTag        string                    `json:"ImageTag"`
	FilePath        string                    `json:"FilePath"`
	Locations       []realtimeengine.Location `json:"Locations"`
	Status          string                    `json:"Status"`
	Vulnerabilities []Vulnerability           `json:"Vulnerabilities"`
}

// ContainerImageResults holds the results of a containers realtime scan.
type ContainerImageResults struct {
	Images []ContainerImage `json:"Images"`
}

type Vulnerability struct {
	CVE         string `json:"CVE"`
	Description string `json:"Description"`
	Severity    string `json:"Severity"`
}
