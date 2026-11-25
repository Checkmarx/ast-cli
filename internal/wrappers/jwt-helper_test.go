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
		existingID := "550e8400-e29b-41d4-a716-446655440000_testuser"
		viper.Set(commonParams.UniqueIDConfigKey, existingID)

		result := GetUniqueID()

		if result != "" {
			assert.Equal(t, existingID, result)
		} else {
			t.Skip("Requires valid auth and 'Checkmarx Developer Assist' or 'Checkmarx OneAssist' license")
		}
	})

	t.Run("generates new unique ID when none exists", func(t *testing.T) {
		// Setup: clear existing ID
		viper.Set(commonParams.UniqueIDConfigKey, "")

		result := GetUniqueID()

		if result == "" {
			t.Skip("Requires valid auth and 'Checkmarx Developer Assist' or 'Checkmarx OneAssist' license")
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

func TestIsValidUniqueIDFormat(t *testing.T) {
	tests := []struct {
		name     string
		uniqueID string
		valid    bool
	}{
		{
			name:     "valid format with UUID and username",
			uniqueID: "550e8400-e29b-41d4-a716-446655440000_testuser",
			valid:    true,
		},
		{
			name:     "invalid - no underscore",
			uniqueID: "550e8400-e29b-41d4-a716-446655440000testuser",
			valid:    false,
		},
		{
			name:     "invalid - empty username",
			uniqueID: "550e8400-e29b-41d4-a716-446655440000_",
			valid:    false,
		},
		{
			name:     "invalid - malformed UUID",
			uniqueID: "550e8400-e29b-41d4-invalid-446655440000_testuser",
			valid:    false,
		},
		{
			name:     "valid - username containing underscore",
			uniqueID: "550e8400-e29b-41d4-a716-446655440000_test_user",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidUniqueIDFormat(tt.uniqueID)
			assert.Equal(t, tt.valid, result, "validation result should match expected")
		})
	}
}

func TestGetSessionIDAlwaysGeneratesNewID(t *testing.T) {
	// Save original value and restore after test
	originalID := viper.GetString(commonParams.SessionIDConfigKey)
	defer viper.Set(commonParams.SessionIDConfigKey, originalID)

	t.Run("generates new session ID when none exists", func(t *testing.T) {
		viper.Set(commonParams.SessionIDConfigKey, "")

		result := GetSessionID()

		assert.Assert(t, result != "", "Should generate new session ID")
		assert.Assert(t, len(result) == 36, "Should be valid UUID format (36 chars)")
		assert.Assert(t, strings.Count(result, "-") == 4, "Should have 4 dashes in UUID")
	})

	t.Run("replaces existing valid session ID", func(t *testing.T) {
		existingID := "550e8400-e29b-41d4-a716-446655440000"
		viper.Set(commonParams.SessionIDConfigKey, existingID)

		result := GetSessionID()

		assert.Assert(t, result != existingID, "Should generate a new ID even when a valid one exists")
		assert.Assert(t, len(result) == 36, "Should generate valid UUID format")
		assert.Equal(t, result, viper.GetString(commonParams.SessionIDConfigKey), "Should store the new ID in config")
	})

	t.Run("replaces invalid session ID", func(t *testing.T) {
		invalidID := "550e8400-e29b-41d4-invalid-446655440000"
		viper.Set(commonParams.SessionIDConfigKey, invalidID)

		result := GetSessionID()

		assert.Assert(t, result != invalidID, "Should regenerate when ID is invalid")
		assert.Assert(t, len(result) == 36, "Should generate valid UUID format")
	})
}

func TestBuildLicenseDetailsFromJWT(t *testing.T) {
	tests := []struct {
		name                string
		allowedEngines      []string
		dastEnabled         bool
		expectedCxOneAssist string
		expectedCxDevAssist string
		expectedDast        string
	}{
		{
			name:                "all features enabled",
			allowedEngines:      []string{"sast", "sca", commonParams.CheckmarxOneAssistType, commonParams.CheckmarxDevAssistType},
			dastEnabled:         true,
			expectedCxOneAssist: "true",
			expectedCxDevAssist: "true",
			expectedDast:        "true",
		},
		{
			name:                "all features enabled - AIProtection",
			allowedEngines:      []string{"sast", "sca", commonParams.CheckmarxOneAssistType, commonParams.AIProtectionType},
			dastEnabled:         true,
			expectedCxOneAssist: "true",
			expectedCxDevAssist: "false",
			expectedDast:        "true",
		},
		{
			name:                "only dev assist enabled",
			allowedEngines:      []string{"sast", commonParams.CheckmarxDevAssistType},
			dastEnabled:         false,
			expectedCxOneAssist: "false",
			expectedCxDevAssist: "true",
			expectedDast:        "false",
		},
		{
			name:                "no assist features enabled",
			allowedEngines:      []string{"sast", "sca", "iac-security"},
			dastEnabled:         false,
			expectedCxOneAssist: "false",
			expectedCxDevAssist: "false",
			expectedDast:        "false",
		},
		{
			name:                "only dast enabled",
			allowedEngines:      []string{"sast"},
			dastEnabled:         true,
			expectedCxOneAssist: "false",
			expectedCxDevAssist: "false",
			expectedDast:        "true",
		},
		{
			name:                "case insensitive matching",
			allowedEngines:      []string{"checkmarx one assist", "checkmarx developer assist"},
			dastEnabled:         false,
			expectedCxOneAssist: "true",
			expectedCxDevAssist: "true",
			expectedDast:        "false",
		},
		{
			name:                "empty allowed engines",
			allowedEngines:      []string{},
			dastEnabled:         false,
			expectedCxOneAssist: "false",
			expectedCxDevAssist: "false",
			expectedDast:        "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a JWT struct with test data
			jwtStruct := &JWTStruct{}
			jwtStruct.AstLicense.LicenseData.AllowedEngines = tt.allowedEngines
			jwtStruct.AstLicense.LicenseData.DastEnabled = tt.dastEnabled

			// Call the function under test
			licenseDetails := buildLicenseDetailsFromJWT(jwtStruct)

			// Assert the results
			assert.Equal(t, tt.expectedCxOneAssist, licenseDetails[commonParams.CxOneAssistEnabledKey],
				"CxOneAssist should be %s", tt.expectedCxOneAssist)
			assert.Equal(t, tt.expectedCxDevAssist, licenseDetails[commonParams.CxDevAssistEnabledKey],
				"CxDevAssist should be %s", tt.expectedCxDevAssist)
			assert.Equal(t, tt.expectedDast, licenseDetails[commonParams.DastEnabledKey],
				"Dast should be %s", tt.expectedDast)
		})
	}
}
