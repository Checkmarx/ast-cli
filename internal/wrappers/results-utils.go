package wrappers

import "encoding/json"

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
