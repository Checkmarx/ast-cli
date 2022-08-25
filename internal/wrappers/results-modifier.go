package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
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

	if strings.HasPrefix(s.Type, "dependency") || strings.HasPrefix(s.Type, "sca-") {
		s.Type = params.ScaType
	}

	s.Status = strings.TrimSpace(s.Status)
	s.State = strings.TrimSpace(s.State)
	s.Severity = strings.TrimSpace(s.Severity)

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
		Column     int    `json:"column"`
		Line       int    `json:"line"`
		Length     int    `json:"length"`
		MethodLine int    `json:"methodLine"`
		FileName   string `json:"fileName,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.Column = 0
	s.Line = 0
	s.Length = 0
	s.MethodLine = 0
	message := "Found negative %v with value %v in file %v"
	if aux.Column >= 0 {
		s.Column = uint(aux.Column)
	} else {
		message = fmt.Sprintf(message, "column", aux.Column, aux.FileName)
		logger.PrintIfVerbose(message)
	}
	if aux.Line >= 0 {
		s.Line = uint(aux.Line)
	} else {
		message = fmt.Sprintf(message, "line", aux.Line, aux.FileName)
		logger.PrintIfVerbose(message)
	}
	if aux.Length >= 0 {
		s.Length = uint(aux.Length)
	} else {
		message = fmt.Sprintf(message, "length", aux.Length, aux.FileName)
		logger.PrintIfVerbose(message)
	}
	if aux.MethodLine >= 0 {
		s.MethodLine = uint(aux.MethodLine)
	} else {
		message = fmt.Sprintf(message, "methodLine", aux.MethodLine, aux.FileName)
		logger.PrintIfVerbose(message)
	}
	return nil
}
