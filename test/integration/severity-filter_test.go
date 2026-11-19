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

const severityFilterCriticalHigh = "critical,high"

func isCriticalOrHigh(severity string) bool {
	switch strings.ToLower(severity) {
	case "critical", "high":
		return true
	}
	return false
}

func TestOssRealtimeScan_SeverityFilter_OnlyAllowedSeveritiesReturned(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "oss-realtime",
		flag(commonParams.SourcesFlag), "data/manifests/package.json",
		flag(commonParams.SeverityFilterFlag), severityFilterCriticalHigh,
	}

	err, bytes := executeCommand(t, args...)
	assert.Nil(t, err, "oss-realtime with --severity-filter should not fail")

	var packages ossrealtime.OssPackageResults
	assert.Nil(t, json.Unmarshal(bytes.Bytes(), &packages), "Failed to unmarshal package results")
	assert.Greater(t, len(packages.Packages), 0, "Every manifest package must still be returned")
	for _, pkg := range packages.Packages {
		if strings.EqualFold(pkg.Status, "Malicious") {
			continue
		}
		for _, v := range pkg.Vulnerabilities {
			assert.True(t, isCriticalOrHigh(v.Severity),
				"package %s returned %s vulnerability outside the filter", pkg.PackageName, v.Severity)
		}
	}
}

func TestOssRealtimeScan_SeverityFilter_InvalidValue_ReturnsError(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "oss-realtime",
		flag(commonParams.SourcesFlag), "data/manifests/package.json",
		flag(commonParams.SeverityFilterFlag), "critical,hgh",
	}

	err, _ := executeCommand(t, args...)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid severity-filter value(s): hgh")
}

func TestContainersRealtimeScan_SeverityFilter_OnlyAllowedSeveritiesReturned(t *testing.T) {
	// The positive Dockerfile case is skipped in this environment until the containers RT API
	// is deployed (see containers-realtime_test.go), so this test uses the same empty Dockerfile
	// the existing success test relies on: it proves the flag is accepted end to end.
	configuration.LoadConfiguration()
	args := []string{
		"scan", "containers-realtime",
		flag(commonParams.SourcesFlag), "data/empty/Dockerfile",
		flag(commonParams.SeverityFilterFlag), severityFilterCriticalHigh,
	}

	err, bytes := executeCommand(t, args...)
	assert.Nil(t, err, "containers-realtime with --severity-filter should not fail")

	var images containersrealtime.ContainerImageResults
	assert.Nil(t, json.Unmarshal(bytes.Bytes(), &images), "Failed to unmarshal image results")
	for _, img := range images.Images {
		if strings.EqualFold(img.Status, "Malicious") {
			continue
		}
		for _, v := range img.Vulnerabilities {
			assert.True(t, isCriticalOrHigh(v.Severity),
				"image %s returned %s vulnerability outside the filter", img.ImageName, v.Severity)
		}
	}
}

func TestSecretsRealtimeScan_SeverityFilter_OnlyAllowedSeveritiesReturned(t *testing.T) {
	args := []string{
		"scan", "secrets-realtime",
		flag(commonParams.SourcesFlag), "data/secret-exposed.txt",
		flag(commonParams.SeverityFilterFlag), severityFilterCriticalHigh,
	}

	err, bytes := executeCommand(t, args...)
	assert.Nil(t, err, "secrets-realtime with --severity-filter should not fail")

	var results []secretsrealtime.SecretsRealtimeResult
	assert.Nil(t, json.Unmarshal(bytes.Bytes(), &results), "Failed to unmarshal secrets results")
	for _, r := range results {
		assert.True(t, isCriticalOrHigh(r.Severity), "secret %s returned %s outside the filter", r.Title, r.Severity)
	}
}

func TestIacRealtimeScan_SeverityFilter_OnlyAllowedSeveritiesReturned(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "iac-realtime",
		flag(commonParams.SourcesFlag), "data/positive1.tf",
		flag(commonParams.EngineFlag), "docker",
		flag(commonParams.SeverityFilterFlag), severityFilterCriticalHigh,
	}

	err, bytes := executeCommand(t, args...)
	assert.Nil(t, err, "iac-realtime with --severity-filter should not fail")

	var results []iacrealtime.IacRealtimeResult
	assert.Nil(t, safeJSONUnmarshal(bytes.Bytes(), &results), "Failed to unmarshal IaC results")
	for _, r := range results {
		assert.True(t, isCriticalOrHigh(r.Severity), "finding %s returned %s outside the filter", r.Title, r.Severity)
	}
}

func TestExecuteASCAScan_SeverityFilter_OnlyAllowedSeveritiesReturned(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "asca",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
		flag(commonParams.SeverityFilterFlag), severityFilterCriticalHigh,
	}

	err, bytes := executeCommand(t, args...)
	assert.Nil(t, err, "asca with --severity-filter should not fail")

	var scanResult grpcs.ScanResult
	assert.Nil(t, json.Unmarshal(bytes.Bytes(), &scanResult), "Failed to unmarshal scan result")
	assert.Nil(t, scanResult.Error)
	for _, d := range scanResult.ScanDetails {
		assert.True(t, isCriticalOrHigh(d.Severity), "rule %s returned %s outside the filter", d.RuleName, d.Severity)
	}
}
