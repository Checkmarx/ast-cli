package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type TelemetryMockWrapper struct {
}

func (t TelemetryMockWrapper) SendAIDataToLog(data *wrappers.DataForAITelemetry) error {
	return nil
}
