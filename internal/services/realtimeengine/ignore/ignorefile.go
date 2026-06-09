// Package ignore creates and updates the realtime-scan ignore file (the lean
// ".checkmarxIgnoredTempList.json" temp-list) that the realtime engines consume via
// --ignored-file-path. It is the write side of the ignore flow; the engines own the read/filter side.
package ignore

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	// defaultDir / defaultFileName form the CLI's default ignore file:
	// <project-root>/.checkmarx/checkmarxIgnoredTempList.json — written under the current working
	// directory (the folder where the agent/Claude runs). The content format matches the IDE temp-list.
	defaultDir      = ".checkmarx"
	defaultFileName = "checkmarxIgnoredTempList.json"

	dirPerm  = 0o750
	filePerm = 0o600
)

// DefaultPath returns the default ignore-file path: ".checkmarx/checkmarxIgnoredTempList.json"
// under the current working directory (the project root where Claude / the agent runs).
func DefaultPath() string {
	return filepath.Join(defaultDir, defaultFileName)
}

// Load reads the ignore file as a list of raw JSON entries. A missing or empty file yields an
// empty list (not an error) so the first ignore creates the file cleanly.
func Load(path string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []json.RawMessage{}, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return []json.RawMessage{}, nil
	}
	var list []json.RawMessage
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// Append adds entry to the list unless an entry with identical content already exists.
// Returns the (possibly extended) list and whether a new entry was added.
func Append(list []json.RawMessage, entry any) ([]json.RawMessage, bool, error) {
	raw, err := json.Marshal(entry)
	if err != nil {
		return list, false, err
	}
	target, err := canonical(raw)
	if err != nil {
		return list, false, err
	}
	for _, existing := range list {
		if c, cErr := canonical(existing); cErr == nil && c == target {
			return list, false, nil // already ignored
		}
	}
	return append(list, json.RawMessage(raw)), true, nil
}

// Remove deletes any entry whose content matches the given one (the revive / review operation).
// Returns the (possibly shortened) list and whether anything was removed.
func Remove(list []json.RawMessage, entry any) ([]json.RawMessage, bool, error) {
	raw, err := json.Marshal(entry)
	if err != nil {
		return list, false, err
	}
	target, err := canonical(raw)
	if err != nil {
		return list, false, err
	}
	out := make([]json.RawMessage, 0, len(list))
	removed := false
	for _, existing := range list {
		if c, cErr := canonical(existing); cErr == nil && c == target {
			removed = true
			continue
		}
		out = append(out, existing)
	}
	return out, removed, nil
}

// Save writes the list as pretty-printed JSON, creating the parent directory if needed.
func Save(path string, list []json.RawMessage) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, filePerm)
}

// canonical normalizes a JSON object so two entries that differ only in key order or whitespace
// compare equal (json.Marshal of a map sorts keys alphabetically).
func canonical(raw []byte) (string, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", err
	}
	out, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
