package ignore

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/containersrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/pkg/errors"
)

// Scan-type identifiers accepted by --scan-type. "sca" is an alias for "oss".
const (
	ScanTypeOSS        = "oss"
	ScanTypeSCA        = "sca"
	ScanTypeSecrets    = "secrets"
	ScanTypeContainers = "containers"
	ScanTypeIaC        = "iac"
	ScanTypeASCA       = "asca"
)

// wrapperKeys are the array fields under which the realtime scans nest their findings. A finding
// payload may be the full scan output, a bare array, or a single finding object.
var wrapperKeys = []string{"Packages", "Images", "scan_details"}

// IsValidScanType reports whether s is an accepted --scan-type value.
func IsValidScanType(s string) bool {
	switch normalizeScanType(s) {
	case ScanTypeOSS, ScanTypeSecrets, ScanTypeContainers, ScanTypeIaC, ScanTypeASCA:
		return true
	default:
		return false
	}
}

func normalizeScanType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == ScanTypeSCA {
		return ScanTypeOSS
	}
	return s
}

// BuildEntries converts a finding payload (single finding, array, or full scan output) into the
// lean ignore entries for the given scan type.
func BuildEntries(scanType string, data []byte) ([]any, error) {
	st := normalizeScanType(scanType)
	findings, err := extractFindings(data)
	if err != nil {
		return nil, err
	}
	if len(findings) == 0 {
		return nil, errors.New("no findings found in --data")
	}
	entries := make([]any, 0, len(findings))
	for _, f := range findings {
		entry, bErr := buildOne(st, f)
		if bErr != nil {
			return nil, bErr
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// extractFindings normalizes the payload into a list of raw finding objects.
func extractFindings(data []byte) ([]json.RawMessage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, errors.New("--data is empty")
	}
	switch data[0] {
	case '[':
		var arr []json.RawMessage
		if err := json.Unmarshal(data, &arr); err != nil {
			return nil, errors.Wrap(err, "parsing findings array")
		}
		return arr, nil
	case '{':
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, errors.Wrap(err, "parsing finding object")
		}
		for _, key := range wrapperKeys {
			if raw, ok := obj[key]; ok {
				var arr []json.RawMessage
				if err := json.Unmarshal(raw, &arr); err != nil {
					return nil, errors.Wrapf(err, "parsing %q array", key)
				}
				return arr, nil
			}
		}
		return []json.RawMessage{data}, nil // a single finding object
	default:
		return nil, errors.New("--data is not valid JSON (expected an object or array)")
	}
}

func buildOne(scanType string, raw json.RawMessage) (any, error) {
	switch scanType {
	case ScanTypeOSS:
		var e ossrealtime.IgnoredPackage
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, errors.Wrap(err, "parsing oss finding")
		}
		if e.PackageManager == "" || e.PackageName == "" || e.PackageVersion == "" {
			return nil, missingFieldsErr(ScanTypeOSS, "PackageManager, PackageName, PackageVersion")
		}
		return e, nil
	case ScanTypeSecrets:
		var e secretsrealtime.IgnoredSecret
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, errors.Wrap(err, "parsing secrets finding")
		}
		if e.Title == "" || e.SecretValue == "" {
			return nil, missingFieldsErr(ScanTypeSecrets, "Title, SecretValue")
		}
		return e, nil
	case ScanTypeContainers:
		var e containersrealtime.IgnoredContainersFinding
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, errors.Wrap(err, "parsing containers finding")
		}
		if e.ImageName == "" || e.ImageTag == "" {
			return nil, missingFieldsErr(ScanTypeContainers, "ImageName, ImageTag")
		}
		return e, nil
	case ScanTypeIaC:
		var e iacrealtime.IgnoredIacFinding
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, errors.Wrap(err, "parsing iac finding")
		}
		if e.Title == "" || e.SimilarityID == "" {
			return nil, missingFieldsErr(ScanTypeIaC, "Title, SimilarityID")
		}
		return e, nil
	case ScanTypeASCA:
		return buildAscaEntry(raw)
	default:
		return nil, errors.Errorf("unsupported scan type %q", scanType)
	}
}

// buildAscaEntry maps an ASCA finding to its ignore entry. The realtime scan emits snake_case
// (file_name/line/rule_id via grpcs.ScanDetail) while the ignore entry is PascalCase
// (FileName/Line/RuleID) — so a direct unmarshal would silently drop FileName/RuleID. We unmarshal
// the scan-output shape first, then fall back to the ignore-entry shape (e.g. for --remove input).
func buildAscaEntry(raw json.RawMessage) (any, error) {
	var sd grpcs.ScanDetail
	_ = json.Unmarshal(raw, &sd)
	fileName, line, ruleID := sd.FileName, sd.Line, sd.RuleID
	if fileName == "" && ruleID == 0 {
		var ig grpcs.AscaIgnoreFinding
		_ = json.Unmarshal(raw, &ig)
		fileName, line, ruleID = ig.FileName, ig.Line, ig.RuleID
	}
	if fileName == "" || ruleID == 0 {
		return nil, missingFieldsErr(ScanTypeASCA, "file_name/FileName, rule_id/RuleID")
	}
	return grpcs.AscaIgnoreFinding{FileName: fileName, Line: line, RuleID: ruleID}, nil
}

func missingFieldsErr(scanType, fields string) error {
	return errors.Errorf("invalid %s finding: missing required field(s): %s", scanType, fields)
}
