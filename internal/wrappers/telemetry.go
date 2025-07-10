package wrappers

import (
	"time"
)

type DataForAITelemetry struct {
	AIProvider      string    `json:"aiProvider"`
	ProblemSeverity string    `json:"problemSeverity"`
	ClickType       string    `json:"clickType"`
	Agent           string    `json:"agent"`
	Engine          string    `json:"engine"`
	Timestamp       time.Time `json:"timestamp"`
}

type TelemetryWrapper interface {
	SendDataToLog(data DataForAITelemetry) error
}
