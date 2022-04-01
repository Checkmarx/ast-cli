package wrappers

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
)

// UnmarshalJSON Function normalizes description to ScanResult
func (s *ScanResult) UnmarshalJSON(data []byte) error {
	type Alias ScanResult
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

	if strings.HasPrefix(s.Type, "infrastructure") {
		s.Type = params.KicsType
	}

	if strings.HasPrefix(s.Type, "dependency") || strings.HasPrefix(s.Type, "sca-container") {
		s.Type = params.ScaType
	}

	if s.Description == "" && s.ScanResultData.Description != "" {
		s.Description = s.ScanResultData.Description
		s.ScanResultData.Description = ""
	}

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
