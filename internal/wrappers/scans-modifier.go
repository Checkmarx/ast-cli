package wrappers

import (
	"bytes"
	"encoding/json"
	"time"
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
