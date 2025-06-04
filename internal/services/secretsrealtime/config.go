package secretsrealtime

type SecretsRealtimeResult struct {
	Title       string `json:"Title"`
	Description string `json:"Description"`
	FilePath    string `json:"FilePath"`
	LineStart   int    `json:"LineStart"`
	LineEnd     int    `json:"LineEnd"`
	StartIndex  int    `json:"StartIndex"`
	EndIndex    int    `json:"EndIndex"`
}
