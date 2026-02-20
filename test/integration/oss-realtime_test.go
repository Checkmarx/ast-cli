//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/stretchr/testify/assert"
)

func TestOssRealtimeScan_RequirementsTxtFile_Success(t *testing.T) {

	configuration.LoadConfiguration()
	_ = executeCmdNilAssertion(t, "Run OSS Realtime scan", "scan", "oss-realtime", "-s", "data/manifests/requirements.txt")
	assert.True(t, validateCacheFileExist())
	defer deleteCacheFile()
}

func TestOssRealtimeScan_PackageJsonFileWithVulnerablePackages_Success(t *testing.T) {
	//t.Skip() // Skip this test for now
	args := []string{
		"scan", "oss-realtime",
		flag(commonParams.SourcesFlag), "data/manifests/package.json",
	}

	err, bytes := executeCommand(t, args...)

	assert.Nil(t, err, "Sending package.json file should not fail")

	var packages ossrealtime.OssPackageResults
	err = json.Unmarshal(bytes.Bytes(), &packages)
	assert.Nil(t, err, "Failed to unmarshal package results")
	assert.Greater(t, len(packages.Packages), 0, "Should return at least one package")
	var hasVulnerabilities bool
	for _, pkg := range packages.Packages {
		if len(pkg.Vulnerabilities) > 0 {
			hasVulnerabilities = true
			break
		}
	}
	assert.True(t, hasVulnerabilities, "At least one package should have vulnerabilities")
}

func TestOssRealtimeScan_PackageJsonFileWithoutPackages_SuccessWithEmptyResponse(t *testing.T) {
	args := []string{
		"scan", "oss-realtime",
		flag(commonParams.SourcesFlag), "data/manifests/no_dep_packageJson/package.json",
	}

	err, bytes := executeCommand(t, args...)

	assert.Nil(t, err, "Sending package.json file should not fail")

	var packages ossrealtime.OssPackageResults
	err = json.Unmarshal(bytes.Bytes(), &packages)
	assert.Nil(t, err, "Failed to unmarshal package results")
	assert.Equal(t, len(packages.Packages), 0, "Should return no packages")
	assert.NotNil(t, packages.Packages)
}

func TestOssRealtimeScan_PackageJsonFile_Success(t *testing.T) {

	configuration.LoadConfiguration()
	_ = executeCmdNilAssertion(t, "Run OSS Realtime scan", "scan", "oss-realtime", "-s", "data/manifests/package.json")
	assert.True(t, validateCacheFileExist())
	defer deleteCacheFile()
}

func TestOssRealtimeScan_PackageJsonFileWithSeverityThreshold_Success(t *testing.T) {

	args := []string{
		"scan", "oss-realtime",
		flag(commonParams.SourcesFlag), "data/manifests/package.json",
		flag(commonParams.SeverityThreshold), "Critical ,High",
	}

	err, bytes := executeCommand(t, args...)

	assert.Nil(t, err, "Sending package.json file should not fail")

	var packages ossrealtime.OssPackageResults
	err = json.Unmarshal(bytes.Bytes(), &packages)
	assert.Nil(t, err, "Failed to unmarshal package results")
	assert.Greater(t, len(packages.Packages), 0, "Should return at least one package")

	for _, pkg := range packages.Packages {
		for _, vul := range pkg.Vulnerabilities {
			assert.True(t, vul.Severity == "Critical" || vul.Severity == "High", "Vulnerability severity should be Critical or High")
		}
	}
}

func TestOssRealtimeScan_PackageJsonFileWithSeverityThresholdContains_ALLPkgData_Success(t *testing.T) {
	// Severity threshold only filters vulnerabilities in OSS, not packages, so all package data should be returned even if severity threshold is set
	args1 := []string{
		"scan", "oss-realtime",
		flag(commonParams.SourcesFlag), "data/manifests/package.json",
	}
	err, bytes := executeCommand(t, args1...)

	assert.Nil(t, err, "Sending package.json file should not fail")

	var packages1 ossrealtime.OssPackageResults
	err = json.Unmarshal(bytes.Bytes(), &packages1)

	args2 := []string{
		"scan", "oss-realtime",
		flag(commonParams.SourcesFlag), "data/manifests/package.json",
		flag(commonParams.SeverityThreshold), "Critical ,High",
	}

	err, bytes2 := executeCommand(t, args2...)

	assert.Nil(t, err, "Sending package.json file should not fail")

	var packages ossrealtime.OssPackageResults
	err = json.Unmarshal(bytes2.Bytes(), &packages)
	assert.Nil(t, err, "Failed to unmarshal package results")
	assert.Equal(t, len(packages1.Packages), len(packages.Packages), "All packages should be returned even with severity threshold")
}

func TestOssRealtimeScan_PackageJsonFileWithSeverityThreshold_ErrorForDifferentSeverity(t *testing.T) {
	args := []string{
		"scan", "oss-realtime",
		flag(commonParams.SourcesFlag), "data/manifests/package.json",
		flag(commonParams.SeverityThreshold), "different",
	}
	err, _ := executeCommand(t, args...)
	assert.NotNil(t, err, "Severity threshold value is not valid")
}

func validateCacheFileExist() bool {
	cacheFilePath := os.TempDir() + "/oss-realtime-cache.json"
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func deleteCacheFile() {
	_ = os.Remove(os.TempDir() + "/oss-realtime-cache.json")
}
