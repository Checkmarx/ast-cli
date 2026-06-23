package mcp

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSecurityMCPURL(t *testing.T) {
	tests := []struct {
		name    string
		issuer  string
		want    string
		wantErr bool
	}{
		{
			name:   "regional iam host maps to ast",
			issuer: "https://eu.iam.checkmarx.net/auth/realms/cx_seg",
			want:   "https://eu.ast.checkmarx.net/api/security-mcp/mcp/cx_seg",
		},
		{
			name:   "us no-prefix iam host maps to ast",
			issuer: "https://iam.checkmarx.net/auth/realms/myorg",
			want:   "https://ast.checkmarx.net/api/security-mcp/mcp/myorg",
		},
		{
			name:   "trailing slash tolerated",
			issuer: "https://deu.iam.checkmarx.net/auth/realms/tenant1/",
			want:   "https://deu.ast.checkmarx.net/api/security-mcp/mcp/tenant1",
		},
		{
			name:    "non-iam host cannot be mapped (use CX_MCP_URL)",
			issuer:  "https://example.com/auth/realms/x",
			wantErr: true,
		},
		{
			// Dev/on-prem hosts like iam-dev.dev.cxast.net do NOT follow the
			// <region>.iam.checkmarx.net pattern, so derivation must fail and the
			// caller is expected to set CX_MCP_URL (see TestDeriveMCPURL_CXMCPURLOverride).
			name:    "dev host is not auto-mappable",
			issuer:  "https://iam-dev.dev.cxast.net/auth/realms/dev_tenant",
			wantErr: true,
		},
		{
			name:    "missing realm segment",
			issuer:  "https://eu.iam.checkmarx.net",
			wantErr: true,
		},
		{
			name:    "empty issuer",
			issuer:  "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildSecurityMCPURL(tt.issuer)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDeriveMCPURL_FlagOverride covers the --mcp-url flag (preferred for MCP
// clients that pass args but not env), which must win even over CX_MCP_URL.
func TestDeriveMCPURL_FlagOverride(t *testing.T) {
	t.Setenv("CX_MCP_URL", "https://from-env.example.com/api/security-mcp/mcp/x")
	got, err := deriveMCPURL("ignored-because-override-set",
		"https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant/")
	assert.NoError(t, err)
	assert.Equal(t, "https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant", got)
}

// TestDeriveMCPURL_CXMCPURLOverride covers the env escape hatch used for dev/on-prem
// environments (e.g. ast-master-components.dev.cxast.net / dev_tenant) whose host
// naming the iam->ast mapping cannot derive.
func TestDeriveMCPURL_CXMCPURLOverride(t *testing.T) {
	t.Setenv("CX_MCP_URL", "https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant/")
	got, err := deriveMCPURL("ignored-because-override-set", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant", got)
}

func TestDeriveMCPURL_NoCredential(t *testing.T) {
	_ = os.Unsetenv("CX_MCP_URL")
	_, err := deriveMCPURL("", "")
	assert.Error(t, err)
}
