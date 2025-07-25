package wrappers

import (
	"testing"

	"gotest.tools/assert"
)

func TestGetDefaultEngines(t *testing.T) {
	tests := []struct {
		name            string
		sscsLicensingV2 bool
		expectedEngines map[string]bool
	}{
		{
			name:            "using new sscs licensing",
			sscsLicensingV2: true,
			expectedEngines: map[string]bool{
				"sast":              true,
				"sca":               true,
				"api-security":      true,
				"iac-security":      true,
				"containers":        true,
				"repository-health": true,
				"secret-detection":  true,
			},
		},
		{
			name:            "using old sscs licensing",
			sscsLicensingV2: false,
			expectedEngines: map[string]bool{
				"sast":               true,
				"sca":                true,
				"api-security":       true,
				"iac-security":       true,
				"containers":         true,
				"scs":                true,
				"enterprise-secrets": true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualEngines := getDefaultEngines(tt.sscsLicensingV2)
			assert.Equal(t, tt.expectedEngines, actualEngines)
		})
	}
}

func TestGetEnabledEngines(t *testing.T) {
	tests := []struct {
		name            string
		sscsLicensingV2 bool
		expectedEngines []string
	}{
		{
			name:            "using new sscs licensing",
			sscsLicensingV2: true,
			expectedEngines: []string{"sast", "sca", "api-security", "iac-security", "containers", "repository-health", "secret-detection"},
		},
		{
			name:            "using old sscs licensing",
			sscsLicensingV2: false,
			expectedEngines: []string{"sast", "sca", "api-security", "iac-security", "containers", "scs", "enterprise-secrets"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualEngines := getDefaultEngines(tt.sscsLicensingV2)
			assert.Equal(t, tt.expectedEngines, actualEngines)
		})
	}
}
