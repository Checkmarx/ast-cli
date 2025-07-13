package wrappers

import (
	"time"
)

type DataForAITelemetry struct {
	AIProvider      string    `json:"aiProvider"`
	ProblemSeverity string    `json:"problemSeverity"`
	Type            string    `json:"type"`
	SubType         string    `json:"subType"`
	Agent           string    `json:"agent"`
	Engine          string    `json:"engine"`
	Timestamp       time.Time `json:"timestamp"`
}

type TelemetryWrapper interface {
	SendAIDataToLog(data DataForAITelemetry) error
}
