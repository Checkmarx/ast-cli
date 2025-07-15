package wrappers

type DataForAITelemetry struct {
	AIProvider      string `json:"aiProvider"`
	ProblemSeverity string `json:"problemSeverity"`
	Type            string `json:"type"`
	SubType         string `json:"subType"`
	Agent           string `json:"agent"`
	Engine          string `json:"engine"`
}

type TelemetryWrapper interface {
	SendAIDataToLog(data *DataForAITelemetry) error
}
