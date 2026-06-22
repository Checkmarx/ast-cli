package sca

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
)

// CheckBashInstall is the entry point for the pre-Bash-tool guardrail. It
// returns ("", "") to allow the command, or (finding, remediation) to block.
// Errors fail open — we never block on infrastructure failures.
//
// Compound commands produce multiple install requests; we scan each and
// return on the first finding (malicious takes precedence over vulnerable).
func (s *Scanner) CheckBashInstall(command, workDir string) (finding, remediation string) {
	s.workDir = workDir
	for _, req := range ParseInstall(command) {
		mal, vuln, err := s.scanRequest(req, workDir)
		if err != nil {
			continue
		}
		if f, r := denyFrom(mal, vuln, workDir); f != "" {
			return f, r
		}
	}
	return "", ""
}

func (s *Scanner) scanRequest(req InstallRequest, workDir string) (malicious, vulnerable []ossrealtime.OssPackage, err error) {
	if req.ManifestRef != "" {
		path := resolveRef(req.ManifestRef, workDir)
		if _, statErr := os.Stat(path); statErr != nil {
			return nil, nil, nil
		}
		return s.ScanFile(path)
	}
	if len(req.Packages) == 0 {
		return nil, nil, nil
	}
	return s.ScanPackages(req.Manager.Format(), req.Packages)
}

// CheckManifestEdit is the entry point for the pre-file-edit guardrail. It
// returns ("", "") to accept the edit, or (finding, remediation) to reject.
//
// Non-manifest paths are a no-op. For manifest paths we diff before/after,
// scan only the newly-added packages, and reject if any are malicious or
// vulnerable.
func (s *Scanner) CheckManifestEdit(filePath string, afterContent []byte, workDir string) (finding, remediation string) {
	s.workDir = workDir
	format, ok := IsManifest(filePath)
	if !ok {
		return "", ""
	}
	before, _ := os.ReadFile(filePath) // missing → empty before
	added, err := AddedPackages(format, before, afterContent)
	if err != nil || len(added) == 0 {
		return "", ""
	}
	mal, vuln, err := s.ScanPackages(format, added)
	if err != nil {
		return "", ""
	}
	return denyFrom(mal, vuln, workDir)
}

// denyFrom builds the (finding, remediation) pair. workDir anchors the
// `cx ignore-vulnerability` suppression command emitted for vulnerable packages
// to the workspace ignore file so the agent writes where the hook later reads.
func denyFrom(malicious, vulnerable []ossrealtime.OssPackage, workDir string) (finding, remediation string) {
	if len(malicious) > 0 {
		return DenyMalicious(malicious)
	}
	if len(vulnerable) > 0 {
		return DenyVulnerable(vulnerable, workDir)
	}
	return "", ""
}
