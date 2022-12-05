package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/gomarkdown/markdown"
)

const (
	message        = "Found negative %v with value %v in file %v"
	column         = "column"
	line           = "line"
	length         = "length"
	methodLine     = "methodLine"
	infrastructure = "infrastructure"
	dependency     = "dependency"
	sca            = "sca-"
)

// UnmarshalJSON Function normalizes description to ScanResult
func (s *ScanResult) UnmarshalJSON(data []byte) error {
	labels := map[string]string{
		params.SastType: params.SastType,
		params.KicsType: params.IacLabel,
		params.ScaType:  params.ScaType,
	}
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

	if strings.HasPrefix(s.Type, infrastructure) {
		s.Type = params.KicsType
	}
	if strings.HasPrefix(s.Type, dependency) || strings.HasPrefix(s.Type, sca) {
		s.Type = params.ScaType
	}

	s.Label = labels[s.Type]
	s.Status = strings.TrimSpace(s.Status)
	s.State = strings.TrimSpace(s.State)
	s.Severity = strings.TrimSpace(s.Severity)

	if s.Description == "" && s.ScanResultData.Description != "" {
		s.Description = s.ScanResultData.Description
		s.ScanResultData.Description = ""
	}

	// Convert markdown description to html description
	s.DescriptionHTML = string(markdown.ToHTML([]byte(s.Description), nil, nil))

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
	s.Column = uintValue(aux.Column, 0, column, aux.FileName)
	s.Line = uintValue(aux.Line, 0, line, aux.FileName)
	s.Length = uintValue(aux.Length, 1, length, aux.FileName)
	s.MethodLine = uintValue(aux.MethodLine, 0, methodLine, aux.FileName)
	s.FileName = aux.FileName

	return nil
}

func uintValue(value, defaultValue int, name, filename string) uint {
	var r = uint(defaultValue)
	if value >= defaultValue {
		r = uint(value)
	} else {
		messageValue := fmt.Sprintf(message, name, value, filename)
		logger.PrintIfVerbose(messageValue)
	}
	return r
}
