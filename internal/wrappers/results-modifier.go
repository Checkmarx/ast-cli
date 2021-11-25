package wrappers

import (
	"encoding/json"
	"fmt"
)

// UnmarshalJSON Function that unmarshal negative columns to 1
func (s *ScanResultNode) UnmarshalJSON(data []byte) error {
	type Alias ScanResultNode
	aux := &struct {
		Column int `json:"column"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	s.Column = 0
	if aux.Column >= 0 {
		s.Column = uint(aux.Column)
	}

	return nil
}

// UnmarshalJSON Convert and create description
func (s *ScanResultData) UnmarshalJSON(data []byte) error {
	type Alias ScanResultData
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Description == "" {
		if aux.ExpectedValue != "" && aux.Value != "" {
			s.Description =
				fmt.Sprintf("File: %s\nValue: %s\nExpected Value: %s", aux.Filename, aux.Value, aux.ExpectedValue)
		}
	}

	return nil
}
