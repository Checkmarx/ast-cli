package wrappers

import "encoding/json"

// UnmarshalJSON Function that unmarshal negative columns to 1
func (s *ScanResultNode) UnmarshalJSON(data []byte) error {
	type Alias ScanResultNode
	aux := &struct {
		Column int `json:"column,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Column <= 0 {
		s.Column = 1
	} else {
		s.Column = uint(aux.Column)
	}

	return nil
}
