package wrappers

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/pkg/errors"
)

// File permission: owner read/write only. Active-mode metadata doesn't hold a
// credential itself, but it does reveal where the user's credential currently
// lives — keep it owner-only to avoid leaking that signal.
const activeModeFilePerm = 0o600

// ActiveModeFilePath returns the absolute path to the active-mode metadata
// file. Derived from the same config directory as the existing yaml so a
// custom --config-file-path is respected.
func ActiveModeFilePath() (string, error) {
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve config file path for active-mode file")
	}
	return filepath.Join(filepath.Dir(configPath), params.ActiveModeFileName), nil
}

// ReadActiveMode returns the currently active session mode — one of
// params.SessionYamlValue, params.SessionLocalValue, or
// params.SessionGlobalValue. Returns ("", nil) if the file does not exist,
// which means "no active session" — every read path falls back to whatever
// the user has set directly (env var or yaml).
func ReadActiveMode() (string, error) {
	path, err := ActiveModeFilePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to read active-mode file")
	}
	mode := strings.TrimSpace(string(data))
	switch mode {
	case params.SessionYamlValue, params.SessionLocalValue, params.SessionGlobalValue, "":
		return mode, nil
	default:
		// Unknown value — treat as no active mode so the CLI doesn't get
		// confused by a corrupt file. Caller can still fall back to defaults.
		return "", nil
	}
}

// WriteActiveMode persists the active session mode. Creates the config
// directory if needed so the first-ever login on a fresh machine works.
func WriteActiveMode(mode string) error {
	if mode != params.SessionYamlValue && mode != params.SessionLocalValue && mode != params.SessionGlobalValue {
		return errors.Errorf("invalid active mode %q: must be %q, %q, or %q",
			mode, params.SessionYamlValue, params.SessionLocalValue, params.SessionGlobalValue)
	}
	path, err := ActiveModeFilePath()
	if err != nil {
		return err
	}
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o700); mkErr != nil {
		return errors.Wrap(mkErr, "failed to create config directory for active-mode file")
	}
	if writeErr := os.WriteFile(path, []byte(mode), activeModeFilePerm); writeErr != nil {
		return errors.Wrap(writeErr, "failed to write active-mode file")
	}
	return nil
}

// ClearActiveMode removes the active-mode file. Returns nil if the file
// already does not exist (logout is idempotent).
func ClearActiveMode() error {
	path, err := ActiveModeFilePath()
	if err != nil {
		return err
	}
	if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
		return errors.Wrap(rmErr, "failed to remove active-mode file")
	}
	return nil
}
