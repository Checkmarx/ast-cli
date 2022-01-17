package wrappers

import (
	"encoding/json"
	"fmt"
)

// UnmarshalJSON Function normalizes description to ScanResult
func (s *ScanResult) UnmarshalJSON(data []byte) error {
	type Alias ScanResult
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Description == "" {
		if aux.ScanResultData.Description != "" {
			aux.Description = aux.ScanResultData.Description
			aux.ScanResultData.Description = ""
		} else if aux.ScanResultData.ExpectedValue != "" && aux.ScanResultData.Value != "" {
			aux.Description = fmt.Sprintf(
				"Value: %s<br />Expected Value: %s",
				aux.ScanResultData.Value,
				aux.ScanResultData.ExpectedValue,
			)
		}
	}

	s.Description = aux.Description
	s.ScanResultData.Description = aux.ScanResultData.Description

	return nil
}

// UnmarshalJSON Function that unmarshal negative columns to 0
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
