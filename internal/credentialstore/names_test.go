package credentialstore

import (
	"encoding/hex"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/stretchr/testify/assert"
)

func TestCanonicalConfigPathDriveLetterEquivalence(t *testing.T) {
	const goosWindows = "windows"
	if runtime.GOOS != goosWindows {
		t.Skip("drive letter normalization applies to Windows only")
	}
	upper := CanonicalConfigPath(`C:\Users\TEST\.checkmarx\checkmarxcli.yaml`)
	lower := CanonicalConfigPath(`c:\users\test\.checkmarx\checkmarxcli.yaml`)
	assert.Equal(t, upper, lower)
	assert.True(t, strings.HasPrefix(upper, "c:"))
}

func TestCanonicalConfigPathRelativeEqualsAbsolute(t *testing.T) {
	tmp := t.TempDir()
	absolute := CanonicalConfigPath(tmp)
	relative := filepath.Join(tmp, "..", filepath.Base(tmp))
	assert.Equal(t, absolute, CanonicalConfigPath(relative))
	assert.Equal(t, absolute, CanonicalConfigPath(absolute))
}

func TestAccountForDeterministic(t *testing.T) {
	canonical := CanonicalConfigPath(filepath.Join(t.TempDir(), "checkmarxcli.yaml"))
	first := AccountFor(canonical, CredentialAPIKey)
	second := AccountFor(canonical, CredentialAPIKey)
	assert.Equal(t, first, second)
}

func TestAccountForDiffersAcrossPathsAndNames(t *testing.T) {
	pathA := CanonicalConfigPath(filepath.Join(t.TempDir(), "a.yaml"))
	pathB := CanonicalConfigPath(filepath.Join(t.TempDir(), "b.yaml"))
	assert.NotEqual(t, AccountFor(pathA, CredentialAPIKey), AccountFor(pathB, CredentialAPIKey))
	assert.NotEqual(t, AccountFor(pathA, CredentialAPIKey), AccountFor(pathA, CredentialClientSecret))
}

func TestAccountForNeverExposesRawPath(t *testing.T) {
	raw := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	canonical := CanonicalConfigPath(raw)
	account := AccountFor(canonical, CredentialAPIKey)
	assert.NotContains(t, account, canonical)
	assert.NotContains(t, account, raw)
	parts := strings.Split(account, accountSeparator)
	assert.Len(t, parts, 2)
	assert.Equal(t, CredentialAPIKey, parts[1])
	decoded, err := hex.DecodeString(parts[0])
	assert.NoError(t, err)
	assert.Len(t, decoded, accountHashBytes)
}

func TestIsValidCredentialName(t *testing.T) {
	assert.True(t, IsValidCredentialName(CredentialAPIKey))
	assert.True(t, IsValidCredentialName(CredentialClientSecret))
	assert.False(t, IsValidCredentialName(""))
	assert.False(t, IsValidCredentialName("api-key "))
	assert.False(t, IsValidCredentialName("API-KEY"))
	assert.False(t, IsValidCredentialName("totally-unrelated"))
}

func TestIsSecret(t *testing.T) {
	assert.True(t, IsSecret(params.AstAPIKey))
	assert.True(t, IsSecret(params.AccessKeySecretConfigKey))
	assert.False(t, IsSecret(params.ConfigFilePathKey))
	assert.False(t, IsSecret(""))
}
