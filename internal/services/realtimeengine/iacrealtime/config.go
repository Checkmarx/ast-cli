package iacrealtime

import "github.com/checkmarx/ast-cli/internal/services/realtimeengine"

type IacRealtimeResult struct {
	SimilarityID string                    `json:"SimilarityID"`
	Title        string                    `json:"Title"`
	Description  string                    `json:"Description"`
	Severity     string                    `json:"Severity"`
	FilePath     string                    `json:"FilePath"`
	Locations    []realtimeengine.Location `json:"Locations"`
}
