package util

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)

	// Parse JSON output
	var result wrappers.JwtClaims
	err = json.Unmarshal(buf.Bytes(), &result)
	assert.NoError(t, err)

	// Verify structured output
	assert.Equal(t, "test-tenant", result.TenantName)
	assert.Equal(t, false, result.DastEnabled)
	assert.ElementsMatch(t, []string{"sast", "sca"}, result.AllowedEngines)
}
