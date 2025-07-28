package iac_realtime

import "github.com/checkmarx/ast-cli/internal/services/realtimeengine"

type IacRealtimeResult struct {
	SimilarityID string                    `json:"SimilarityID"`
	Title        string                    `json:"Title"`
	Description  string                    `json:"Description"`
	Severity     string                    `json:"Severity"`
	FilePath     string                    `json:"FilePath"`
	Locations    []realtimeengine.Location `json:"locations"`
}

var Severities = map[string]string{
	"critical": "Critical",
	"high":     "High",
	"medium":   "Medium",
	"low":      "Low",
	"info":     "Info",
	"unknown":  "Unknown",
}
