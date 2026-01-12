package util

import (
	"bytes"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestLicenseCommandDefaultListFormat(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{
		DastEnabled: true,
	}

	cmd := NewLicenseCommand(mockJWT)
	
	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	assert.NilError(t, err, "License command should run with no errors")
	
	output := buf.String()
	
	// Verify output contains expected fields in list format
	assert.Assert(t, strings.Contains(output, "TenantName"), "Output should contain TenantName")
	assert.Assert(t, strings.Contains(output, "test-tenant"), "Output should contain test-tenant value")
	assert.Assert(t, strings.Contains(output, "DastEnabled"), "Output should contain DastEnabled")
	assert.Assert(t, strings.Contains(output, "true"), "Output should contain true for DastEnabled")
	assert.Assert(t, strings.Contains(output, "AllowedEngines"), "Output should contain AllowedEngines")
}

