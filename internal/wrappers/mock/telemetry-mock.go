package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type TelemetryMockWrapper struct {
}

func (t TelemetryMockWrapper) SendDataToLog(data wrappers.DataForAITelemetry) error {
	return nil
}

//func NewTelemetryMockWrapper() *TelemetryMockWrapper {
//	return &TelemetryMockWrapper{}
//}
