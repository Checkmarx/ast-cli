package wrappers

import (
	"strings"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestGetDefaultEngines(t *testing.T) {
	tests := []struct {
		name            string
		scsLicensingV2  bool
		expectedEngines map[string]bool
	}{
		{
			name:           "using new sscs licensing",
			scsLicensingV2: true,
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
			name:           "using old sscs licensing",
			scsLicensingV2: false,
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
			actualEngines := getDefaultEngines(tt.scsLicensingV2)
			assert.Equal(t, tt.expectedEngines, actualEngines)
		})
	}
}

func TestGetEnabledEngines(t *testing.T) {
	tests := []struct {
		name            string
		scsLicensingV2  bool
		expectedEngines []string
	}{
		{
			name:            "using new sscs licensing",
			scsLicensingV2:  true,
			expectedEngines: []string{"sast", "sca", "api-security", "iac-security", "containers", "repository-health", "secret-detection"},
		},
		{
			name:            "using old sscs licensing",
			scsLicensingV2:  false,
			expectedEngines: []string{"sast", "sca", "api-security", "iac-security", "containers", "scs", "enterprise-secrets"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualEngines := getDefaultEngines(tt.scsLicensingV2)
			assert.Equal(t, tt.expectedEngines, actualEngines)
		})
	}
}

func TestGetUniqueID(t *testing.T) {
	// Save original value and restore after test
	originalID := viper.GetString(commonParams.UniqueIDConfigKey)
	defer viper.Set(commonParams.UniqueIDConfigKey, originalID)

	t.Run("returns existing unique ID from config", func(t *testing.T) {
		// Setup: set existing ID
		existingID := "test-uuid-456_testuser"
		viper.Set(commonParams.UniqueIDConfigKey, existingID)

		result := GetUniqueID()

		if result != "" {
			assert.Equal(t, existingID, result)
		} else {
			t.Skip("Requires valid auth and 'Checkmarx Developer Assist' license")
		}
	})

	t.Run("generates new unique ID when none exists", func(t *testing.T) {
		// Setup: clear existing ID
		viper.Set(commonParams.UniqueIDConfigKey, "")

		result := GetUniqueID()

		if result == "" {
			t.Skip("Requires valid auth and 'Checkmarx Developer Assist' license")
			return
		}

		// Verify format: UUID_username
		assert.Assert(t, strings.Contains(result, "_"), "Should have UUID_username format")
		assert.Assert(t, len(result) > 36, "Should contain UUID and username")

		// Verify no backslash (Windows domain stripped)
		parts := strings.Split(result, "_")
		assert.Assert(t, len(parts) >= 2, "Should have at least 2 parts")
		assert.Assert(t, !strings.Contains(parts[1], "\\"), "Username should not contain backslash")
	})
}
