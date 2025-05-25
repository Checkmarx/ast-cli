//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/stretchr/testify/assert"
)

func TestOssRealtimeScan_RequirementsTxtFile_Success(t *testing.T) {
	t.Skip() // Skip this test for now
	configuration.LoadConfiguration()
	_ = executeCmdNilAssertion(t, "Run OSS Realtime scan", "scan", "oss-realtime", "-s", "data/manifests/requirements.txt")
	assert.True(t, validateCacheFileExist())
	defer deleteCacheFile()
}

func TestOssRealtimeScan_PackageJsonFileWithVulnerablePackages_Success(t *testing.T) {
	t.Skip() // Skip this test for now
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

func TestOssRealtimeScan_PackageJsonFile_Success(t *testing.T) {
	t.Skip() // Skip this test for now
	configuration.LoadConfiguration()
	_ = executeCmdNilAssertion(t, "Run OSS Realtime scan", "scan", "oss-realtime", "-s", "data/manifests/package.json")
	assert.True(t, validateCacheFileExist())
	defer deleteCacheFile()
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
