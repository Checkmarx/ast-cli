//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/containersrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/stretchr/testify/assert"
)

const (
	highRank       = 4
	criticalRank   = 5
	thresholdFlag  = "--severity-threshold"
	invalidMessage = "invalid severity-threshold value"
)

func severityRank(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return criticalRank
	case "high":
		return highRank
	case "medium":
		return 3
	case "low":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// atOrAbove is true for severities at or above rank; unrecognised severities are kept by the CLI.
func atOrAbove(severity string, rank int) bool {
	r := severityRank(severity)
	return r == 0 || r >= rank
}

func runRealtimeScan(t *testing.T, command, source string, extra ...string) []byte {
	_ = configuration.LoadConfiguration()
	args := append([]string{"scan", command, flag(commonParams.SourcesFlag), source}, extra...)
	err, out := executeCommand(t, args...)
	assert.Nil(t, err, "%s %v should not fail", command, extra)
	return out.Bytes()
}

func runRealtimeScanExpectingError(t *testing.T, command, source string, extra ...string) error {
	_ = configuration.LoadConfiguration()
	args := append([]string{"scan", command, flag(commonParams.SourcesFlag), source}, extra...)
	err, _ := executeCommand(t, args...)
	assert.NotNil(t, err)
	return err
}

func ossResults(t *testing.T, extra ...string) ossrealtime.OssPackageResults {
	var results ossrealtime.OssPackageResults
	assert.Nil(t, json.Unmarshal(runRealtimeScan(t, "oss-realtime", "data/manifests/package.json", extra...), &results))
	return results
}

func ossVulnerabilityCount(results ossrealtime.OssPackageResults) int {
	count := 0
	for i := range results.Packages {
		count += len(results.Packages[i].Vulnerabilities)
	}
	return count
}

func TestOssRealtimeScan_SeverityThreshold_High_ReturnsSubsetAtOrAboveHigh(t *testing.T) {
	all := ossResults(t)
	filtered := ossResults(t, thresholdFlag, "high")

	assert.Greater(t, ossVulnerabilityCount(all), 0, "the fixture manifest must have vulnerabilities")
	expected := 0
	for i := range all.Packages {
		if atOrAbove(all.Packages[i].Status, highRank) {
			expected++
		}
	}
	assert.Equal(t, expected, len(filtered.Packages), "only packages with a status below High must be dropped")
	assert.LessOrEqual(t, ossVulnerabilityCount(filtered), ossVulnerabilityCount(all))
	for _, pkg := range filtered.Packages {
		assert.True(t, atOrAbove(pkg.Status, highRank), "package %s returned with status %s", pkg.PackageName, pkg.Status)
		if strings.EqualFold(pkg.Status, "Malicious") {
			continue
		}
		for _, v := range pkg.Vulnerabilities {
			assert.True(t, atOrAbove(v.Severity, highRank), "package %s returned %s below the threshold", pkg.PackageName, v.Severity)
		}
	}
}

func TestOssRealtimeScan_SeverityThreshold_Info_ReturnsEverything(t *testing.T) {
	all := ossResults(t)
	assert.Greater(t, ossVulnerabilityCount(all), 0)
	assert.Equal(t, ossVulnerabilityCount(all), ossVulnerabilityCount(ossResults(t, thresholdFlag, "info")))
}

func TestOssRealtimeScan_SeverityThreshold_InvalidValues_ReturnError(t *testing.T) {
	for _, value := range []string{"hgh", "critical,high"} {
		err := runRealtimeScanExpectingError(t, "oss-realtime", "data/manifests/package.json", thresholdFlag, value)
		assert.Contains(t, err.Error(), invalidMessage)
	}
}

func TestContainersRealtimeScan_SeverityThreshold_FlagAccepted(t *testing.T) {
	// The containers realtime API is not deployed in this environment, so this checks the flag only; the
	// unit tests assert the filtered content with mocked responses.
	var images containersrealtime.ContainerImageResults
	assert.Nil(t, json.Unmarshal(runRealtimeScan(t, "containers-realtime", "data/empty/Dockerfile", thresholdFlag, "high"), &images))

	err := runRealtimeScanExpectingError(t, "containers-realtime", "data/empty/Dockerfile", thresholdFlag, "critical,high")
	assert.Contains(t, err.Error(), invalidMessage)
}

func secretsResults(t *testing.T, extra ...string) []secretsrealtime.SecretsRealtimeResult {
	var results []secretsrealtime.SecretsRealtimeResult
	assert.Nil(t, json.Unmarshal(runRealtimeScan(t, "secrets-realtime", "data/secret-exposed.txt", extra...), &results))
	return results
}

func TestSecretsRealtimeScan_SeverityThreshold_ReturnsSubsetAtOrAboveThreshold(t *testing.T) {
	all := secretsResults(t)
	assert.NotEmpty(t, all, "the fixture file contains a secret")
	assert.Len(t, secretsResults(t, thresholdFlag, "info"), len(all))

	critical := secretsResults(t, thresholdFlag, "critical")
	assert.LessOrEqual(t, len(critical), len(all))
	for _, r := range critical {
		assert.True(t, atOrAbove(r.Severity, criticalRank), "secret %s returned %s below the threshold", r.Title, r.Severity)
	}
}

func iacResults(t *testing.T, extra ...string) []iacrealtime.IacRealtimeResult {
	extra = append([]string{flag(commonParams.EngineFlag), "docker"}, extra...)
	var results []iacrealtime.IacRealtimeResult
	assert.Nil(t, safeJSONUnmarshal(runRealtimeScan(t, "iac-realtime", "data/positive1.tf", extra...), &results))
	return results
}

func TestIacRealtimeScan_SeverityThreshold_ReturnsSubsetAtOrAboveThreshold(t *testing.T) {
	all := iacResults(t)
	assert.NotEmpty(t, all, "the fixture file has IaC findings")

	high := iacResults(t, thresholdFlag, "high")
	assert.LessOrEqual(t, len(high), len(all))
	for _, r := range high {
		assert.True(t, atOrAbove(r.Severity, highRank), "finding %s returned %s below the threshold", r.Title, r.Severity)
	}
}

func ascaResults(t *testing.T, extra ...string) grpcs.ScanResult {
	extra = append([]string{flag(commonParams.AgentFlag), commonParams.DefaultAgent}, extra...)
	var result grpcs.ScanResult
	assert.Nil(t, json.Unmarshal(runRealtimeScan(t, "asca", "data/python-vul-file.py", extra...), &result))
	assert.Nil(t, result.Error)
	return result
}

func TestExecuteASCAScan_SeverityThreshold_ReturnsSubsetAtOrAboveThreshold(t *testing.T) {
	all := ascaResults(t)
	assert.NotEmpty(t, all.ScanDetails, "the fixture file has ASCA findings")

	high := ascaResults(t, thresholdFlag, "high")
	assert.LessOrEqual(t, len(high.ScanDetails), len(all.ScanDetails))
	for _, d := range high.ScanDetails {
		assert.True(t, atOrAbove(d.Severity, highRank), "rule %s returned %s below the threshold", d.RuleName, d.Severity)
	}
}
