package secretsrealtime

import "github.com/checkmarx/ast-cli/internal/services/realtimeengine"

type SecretsRealtimeResult struct {
	Title       string                    `json:"Title"`
	Description string                    `json:"Description"`
	FilePath    string                    `json:"FilePath"`
	Severity    string                    `json:"Severity"`
	Locations   []realtimeengine.Location `json:"Locations"`
}
