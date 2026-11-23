package wrappers

// EventAttributes is a flexible map for telemetry event key-value pairs.
type EventAttributes map[string]string

type DataForAITelemetry struct {
	AIProvider       string           `json:"aiProvider"`
	ProblemSeverity  string           `json:"problemSeverity"`
	Type             string           `json:"type"`
	SubType          string           `json:"subType"`
	Agent            string           `json:"agent"`
	AgentVersion     string           `json:"agentVersion"`
	CxPluginVersion  string           `json:"cxPluginVersion"`
	PluginType       string           `json:"pluginType"`
	Engine           string           `json:"engine"`
	ScanType         string           `json:"scanType"`
	Status           string           `json:"status"`
	StatusResult     string           `json:"statusResult"`
	APIVersion       string           `json:"apiVersion,omitempty"`
	TotalCount       int              `json:"totalCount"`
	RepoURL          string           `json:"repoUrl"`
	UniqueID         string           `json:"uniqueId"`
	SessionID        string           `json:"sessionId"`
	EventAttributes  *EventAttributes `json:"eventAttributes"`
	AiAgentSessionId string           `json:"aiAgentSessionId"`
}

type TelemetryWrapper interface {
	SendAIDataToLog(data *DataForAITelemetry) error
}
