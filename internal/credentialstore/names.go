package credentialstore

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"runtime"
	"strings"
)

// KeyringServiceName is the service name used for all keyring entries.
const KeyringServiceName = "checkmarx-ast-cli"

const (
	accountHashBytes = 16
	accountSeparator = ":"
)

// CanonicalConfigPath returns an absolute cleaned path, lowercased on Windows
// so casing variants of the same file map to one keyring account.
func CanonicalConfigPath(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	return normalizeWindowsPath(abs)
}

// normalizeWindowsPath lowercases the whole path on Windows (NTFS is
// case-insensitive, including UNC shares) and leaves paths on other
// platforms untouched.
func normalizeWindowsPath(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}
	return strings.ToLower(path)
}

// AccountFor derives the keyring account from the canonical config path and logical name.
func AccountFor(canonicalConfigPath, credentialName string) string {
	sum := sha256.Sum256([]byte(canonicalConfigPath))
	return hex.EncodeToString(sum[:accountHashBytes]) + accountSeparator + credentialName
}
