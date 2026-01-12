package util

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/require"
)

func TestLicenseCommandDefaultListFormat(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{
		TenantName:     "test-tenant",
		DastEnabled:    false,
		AllowedEngines: []string{"sast", "sca"},
	}

	cmd := NewLicenseCommand(mockJWT)
	cmd.SetArgs([]string{"--" + params.FormatFlag, "json"})

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse JSON output
	var result wrappers.JwtClaims
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Verify structured output
	require.Equal(t, "test-tenant", result.TenantName)
	require.Equal(t, false, result.DastEnabled)
	require.ElementsMatch(t, result.AllowedEngines, []string{"sast", "sca"})
}
