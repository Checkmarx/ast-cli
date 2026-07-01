package wrappers

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// File permission: owner read/write only. The global session file holds a
// refresh token; readable group/world would be a credential leak.
const sessionGlobalFilePerm = 0o600

// SessionGlobalFilePath returns the absolute path to the global session file.
// Derived from the same config directory as the existing yaml (so a custom
// --config-file-path is respected), with the filename swapped to
// SessionGlobalFileName.
func SessionGlobalFilePath() (string, error) {
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve config file path for global session file")
	}
	dir := filepath.Dir(configPath)
	return filepath.Join(dir, params.SessionGlobalFileName), nil
}

// ReadSessionGlobal returns the refresh token persisted by --session global
// mode. Returns ("", nil) if the file does not exist — that just means the
// user has not logged in via global mode. Any other I/O error is surfaced.
func ReadSessionGlobal() (string, error) {
	path, err := SessionGlobalFilePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to read global session file")
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteSessionGlobal writes the refresh token to the global session file with
// owner-only permissions. Creates the parent directory if necessary so the
// first-ever --session global login on a fresh machine works.
func WriteSessionGlobal(refreshToken string) error {
	path, err := SessionGlobalFilePath()
	if err != nil {
		return err
	}
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o700); mkErr != nil {
		return errors.Wrap(mkErr, "failed to create config directory for global session file")
	}
	if writeErr := os.WriteFile(path, []byte(refreshToken), sessionGlobalFilePerm); writeErr != nil {
		return errors.Wrap(writeErr, "failed to write global session file")
	}
	return nil
}

// ClearSessionGlobal removes the global session file. Returns nil if the file
// already does not exist (logout is idempotent — running it twice is fine).
func ClearSessionGlobal() error {
	path, err := SessionGlobalFilePath()
	if err != nil {
		return err
	}
	if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
		return errors.Wrap(rmErr, "failed to remove global session file")
	}
	return nil
}

// LoadActiveCredential makes the refresh token from the active session mode
// available to viper, so every CLI command's existing cx_apikey lookup
// resolves to the right credential — without any precedence chain.
//
// The active-mode metadata file (~/.checkmarx/active_mode) tells us where
// the user's current credential lives:
//
//   - "yaml":   read cx_apikey from the yaml config file; viper.Set it so it
//     wins over any stale CX_APIKEY env var left over from a
//     previous --session local invocation
//   - "local":  no action — env-binding (viper.BindEnv) already gives env the
//     right precedence; we want the user's current shell env to
//     win
//   - "global": read the dedicated global file; viper.Set it so it wins over
//     any stale yaml or env value
//   - "":       no active session — viper's natural precedence applies
//     (env > yaml). Backward-compatible with users who set
//     CX_APIKEY directly or who logged in with the previous CLI.
//
// Called once at startup from main, after configuration.LoadConfiguration.
func LoadActiveCredential() {
	mode, err := ReadActiveMode()
	if err != nil || mode == "" {
		return
	}
	switch mode {
	case params.SessionGlobalValue:
		rt, err := ReadSessionGlobal()
		if err == nil && rt != "" {
			viper.Set(params.AstAPIKey, rt)
		}
	case params.SessionYamlValue:
		// Yaml's cx_apikey is already loaded by configuration.LoadConfiguration,
		// but env-binding would override it if a stale CX_APIKEY is set in
		// this shell. viper.Set with the yaml value forces yaml to win.
		yamlRT := ReadYamlAPIKey()
		if yamlRT != "" {
			viper.Set(params.AstAPIKey, yamlRT)
		}
	case params.SessionLocalValue:
		// Env binding already gives the current shell's CX_APIKEY the right
		// precedence (env > config file in viper). Nothing to do here.
		// If the user is in a shell that didn't run the --session local
		// login, env will be empty and the command will surface a clear
		// "not authenticated" error.
	}
}

// ReadYamlAPIKey reads cx_apikey directly from the yaml config file, bypassing
// viper's env-first precedence. Used by LoadActiveCredential to force yaml to
// win when the active mode is "yaml" but a stale CX_APIKEY env var exists, and
// by the auth login/logout nuke phase to revoke whatever yaml actually holds
// (not what viper currently resolves to, which could be a stale env var).
func ReadYamlAPIKey() string {
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		return ""
	}
	yamlConfig, err := configuration.LoadConfig(configPath)
	if err != nil {
		return ""
	}
	if v, ok := yamlConfig[params.AstAPIKey].(string); ok {
		return v
	}
	return ""
}
