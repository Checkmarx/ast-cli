package wrappers

import (
	"bytes"
	"encoding/json"
	"time"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
)

// UnmarshalJSON Function normalizes description to ScanResult
func (s *ScanResponseModel) UnmarshalJSON(data []byte) error {
	type Alias ScanResponseModel
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	reader := bytes.NewReader(data)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	if err := decoder.Decode(&aux); err != nil {
		return err
	}

	s.CreatedAt = s.CreatedAt.In(time.Local)

	return nil
}

func (s *ScanResponseModel) ReplaceMicroEnginesWithSCS() {
	for i, status := range s.StatusDetails {
		if status.Name == commonParams.MicroEnginesType {
			s.StatusDetails[i].Name = commonParams.ScsType
		}
	}
	for i, engine := range s.Engines {
		if engine == commonParams.MicroEnginesType {
			s.Engines[i] = commonParams.ScsType
		}
	}
}
