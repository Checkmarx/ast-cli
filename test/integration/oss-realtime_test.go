//go:build integration

package integration

import (
	"os"
	"testing"

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
