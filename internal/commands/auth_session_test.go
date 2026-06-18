//go:build !integration

package commands

import (
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
)

func TestValidateSessionFlag(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		wantErr  bool
		errMatch string // substring expected in error message
	}{
		{name: "empty is valid (default yaml mode)", value: "", wantErr: false},
		{name: "local is valid", value: params.SessionLocalValue, wantErr: false},
		{name: "global is valid", value: params.SessionGlobalValue, wantErr: false},
		{name: "rejects unknown value", value: "yolo", wantErr: true, errMatch: "invalid --session value"},
		{name: "rejects empty-looking but not equal", value: "  ", wantErr: true, errMatch: "invalid --session value"},
		{name: "rejects case mismatch", value: "Local", wantErr: true, errMatch: "invalid --session value"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSessionFlag(tc.value)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for value %q, got nil", tc.value)
					return
				}
				if tc.errMatch != "" && !strings.Contains(err.Error(), tc.errMatch) {
					t.Errorf("expected error containing %q, got %q", tc.errMatch, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("expected no error for value %q, got: %v", tc.value, err)
			}
		})
	}
}
