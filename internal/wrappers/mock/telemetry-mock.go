package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type TelemetryMockWrapper struct {
	CustomSendAIDataToLog func(data *wrappers.DataForAITelemetry) error
}

func (t TelemetryMockWrapper) SendAIDataToLog(data *wrappers.DataForAITelemetry) error {
	if t.CustomSendAIDataToLog != nil {
		return t.CustomSendAIDataToLog(data)
	}
	return nil
}
