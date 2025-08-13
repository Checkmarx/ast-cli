package iacrealtime

import "github.com/checkmarx/ast-cli/internal/services/realtimeengine"

type IacRealtimeResult struct {
	SimilarityID  string                    `json:"SimilarityID"`
	Title         string                    `json:"Title"`
	Description   string                    `json:"Description"`
	ExpectedValue string                    `json:"ExpectedValue"`
	ActualValue   string                    `json:"ActualValue"`
	Severity      string                    `json:"Severity"`
	FilePath      string                    `json:"FilePath"`
	Locations     []realtimeengine.Location `json:"Locations"`
}

type IgnoredIacFinding struct {
	Title        string `json:"Title"`
	SimilarityID string `json:"SimilarityID"`
}
