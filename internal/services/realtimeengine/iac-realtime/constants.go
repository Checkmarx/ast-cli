package iac_realtime

const (
	ContainerPath            = "/path"
	ContainerFormat          = "json"
	ContainerTempDirPattern  = "iac-realtime"
	KicsContainerPrefix      = "cli-iac-realtime-"
	ContainerResultsFileName = "results.json"
)

var KicsErrorCodes = []string{"60", "50", "40", "30", "20"}

type LineIndex struct {
	Start int
	End   int
}

var Severities = map[string]string{
	"critical": "Critical",
	"high":     "High",
	"medium":   "Medium",
	"low":      "Low",
	"info":     "Info",
	"unknown":  "Unknown",
}
