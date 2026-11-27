//go:build !integration

package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/containersrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const (
	invalidSeverityThresholdList = "critical,high"
	invalidSeverityThresholdWord = "hgh"
	invalidSeverityThresholdMsg  = "invalid severity-threshold value"
	singleLevelMsg               = "single level: critical, high, medium, low or info"
)

func enableRealtimeFlag() {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
}

func ossMockWithHighAndLow(status string) *mock.RealtimeScannerMockWrapper {
	return &mock.RealtimeScannerMockWrapper{
		CustomScanPackages: func(req *wrappers.RealtimeScannerPackageRequest) (*wrappers.RealtimeScannerPackageResponse, error) {
			var resp wrappers.RealtimeScannerPackageResponse
			for _, p := range req.Packages {
				resp.Packages = append(resp.Packages, wrappers.RealtimeScannerResults{
					PackageManager: p.PackageManager,
					PackageName:    p.PackageName,
					Version:        p.Version,
					Status:         status,
					Vulnerabilities: []wrappers.RealtimeScannerVulnerability{
						{CVE: "CVE-H", Severity: "High"},
						{CVE: "CVE-L", Severity: "Low"},
					},
				})
			}
			return &resp, nil
		},
	}
}

func runOssRealtimeWithThreshold(t *testing.T, threshold string) ossrealtime.OssPackageResults {
	return runOssRealtime(t, "High", threshold)
}

func runOssRealtime(t *testing.T, status, threshold string) ossrealtime.OssPackageResults {
	enableRealtimeFlag()
	// A fresh temp folder keeps the OSS cache of earlier runs from replacing the mocked response.
	cacheDir := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, cacheDir)
	}
	out := bytes.NewBuffer(nil)
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	cmd.Flags().String(commonParams.SourcesFlag, "data/manifests/package.json", "")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "")
	cmd.Flags().String(commonParams.SeverityThresholdFlag, threshold, "")

	err := RunScanOssRealtimeCommand(ossMockWithHighAndLow(status), &mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{})(cmd, nil)
	assert.NoError(t, err)

	var results ossrealtime.OssPackageResults
	assert.NoError(t, json.Unmarshal(out.Bytes(), &results))
	return results
}

func cvesOf(vulns []ossrealtime.Vulnerability) []string {
	var cves []string
	for _, v := range vulns {
		cves = append(cves, v.CVE)
	}
	return cves
}

func TestOssRealtime_SeverityThreshold_ValidValue_Success(t *testing.T) {
	enableRealtimeFlag()
	execCmdNilAssertion(t, "scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-threshold", " High ")
}

func TestOssRealtime_SeverityThreshold_EmptyValue_TreatedAsNoThreshold(t *testing.T) {
	enableRealtimeFlag()
	execCmdNilAssertion(t, "scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-threshold", "")
}

func TestOssRealtime_SeverityThreshold_InvalidValues_Fail(t *testing.T) {
	for _, value := range []string{invalidSeverityThresholdWord, invalidSeverityThresholdList} {
		enableRealtimeFlag()
		err := execCmdNotNilAssertion(t, "scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-threshold", value)
		assert.Contains(t, err.Error(), invalidSeverityThresholdMsg)
		assert.Contains(t, err.Error(), singleLevelMsg)
	}
}

func scannedPackages(results ossrealtime.OssPackageResults) []ossrealtime.OssPackage {
	var scanned []ossrealtime.OssPackage
	for i := range results.Packages {
		if len(results.Packages[i].Vulnerabilities) > 0 {
			scanned = append(scanned, results.Packages[i])
		}
	}
	return scanned
}

func TestOssRealtime_SeverityThreshold_High_KeepsOnlyHighAndAbove(t *testing.T) {
	all := runOssRealtimeWithThreshold(t, "")
	filtered := runOssRealtimeWithThreshold(t, "high")

	assert.Len(t, filtered.Packages, len(all.Packages), "every package entry must be kept")
	assert.NotEmpty(t, scannedPackages(filtered))
	for _, p := range scannedPackages(filtered) {
		assert.Equal(t, []string{"CVE-H"}, cvesOf(p.Vulnerabilities), "package %s", p.PackageName)
	}
}

func TestOssRealtime_SeverityThreshold_Low_KeepsLowAndAbove(t *testing.T) {
	filtered := runOssRealtimeWithThreshold(t, "low")

	assert.NotEmpty(t, scannedPackages(filtered))
	for _, p := range scannedPackages(filtered) {
		assert.Equal(t, []string{"CVE-H", "CVE-L"}, cvesOf(p.Vulnerabilities), "package %s", p.PackageName)
	}
}

func TestOssRealtime_SeverityThreshold_Critical_DropsPackagesWithLowerStatus(t *testing.T) {
	all := runOssRealtimeWithThreshold(t, "")
	filtered := runOssRealtimeWithThreshold(t, "critical")

	assert.NotEmpty(t, scannedPackages(all), "the mock returns vulnerabilities for scanned packages")
	assert.Len(t, filtered.Packages, len(all.Packages)-len(scannedPackages(all)), "packages with status High must be dropped")
	for i := range filtered.Packages {
		assert.NotEqual(t, "High", filtered.Packages[i].Status, "package %s", filtered.Packages[i].PackageName)
	}
}

func TestOssRealtime_SeverityThreshold_Critical_ReturnsMaliciousPackagesUnchanged(t *testing.T) {
	all := runOssRealtime(t, "Malicious", "")
	filtered := runOssRealtime(t, "Malicious", "critical")

	assert.NotEmpty(t, scannedPackages(all))
	assert.Equal(t, all.Packages, filtered.Packages)
}

func TestOssRealtime_SeverityThreshold_Omitted_ReturnsAllSeverities(t *testing.T) {
	all := runOssRealtimeWithThreshold(t, "")

	assert.NotEmpty(t, scannedPackages(all))
	for _, p := range scannedPackages(all) {
		assert.Equal(t, []string{"CVE-H", "CVE-L"}, cvesOf(p.Vulnerabilities), "package %s", p.PackageName)
	}
}

func runContainersRealtimeWithThreshold(t *testing.T, threshold string) containersrealtime.ContainerImageResults {
	enableRealtimeFlag()
	rt := &mock.RealtimeScannerMockWrapper{
		CustomScanImages: func(req *wrappers.ContainerImageRequest) (*wrappers.ContainerImageResponse, error) {
			var resp wrappers.ContainerImageResponse
			for _, img := range req.Images {
				resp.Images = append(resp.Images, wrappers.ContainerImageResponseItem{
					ImageName: img.ImageName,
					ImageTag:  img.ImageTag,
					Status:    "High",
					Vulnerabilities: []wrappers.ContainerImageVulnerability{
						{CVE: "CVE-H", Severity: "High"},
						{CVE: "CVE-L", Severity: "Low"},
					},
				})
			}
			return &resp, nil
		},
	}
	out := bytes.NewBuffer(nil)
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	cmd.Flags().String(commonParams.SourcesFlag, "data/Dockerfile", "")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "")
	cmd.Flags().String(commonParams.SeverityThresholdFlag, threshold, "")

	err := RunScanContainersRealtimeCommand(rt, &mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{})(cmd, nil)
	assert.NoError(t, err)

	var results containersrealtime.ContainerImageResults
	assert.NoError(t, json.Unmarshal(out.Bytes(), &results))
	return results
}

func TestContainersRealtime_SeverityThreshold_ValidValue_Success(t *testing.T) {
	enableRealtimeFlag()
	execCmdNilAssertion(t, "scan", "containers-realtime", "-s", "data/Dockerfile", "--severity-threshold", "high")
}

func TestContainersRealtime_SeverityThreshold_InvalidValue_Fails(t *testing.T) {
	enableRealtimeFlag()
	err := execCmdNotNilAssertion(t, "scan", "containers-realtime", "-s", "data/Dockerfile", "--severity-threshold", invalidSeverityThresholdList)
	assert.Contains(t, err.Error(), invalidSeverityThresholdMsg)
}

func TestContainersRealtime_SeverityThreshold_High_KeepsOnlyHighAndAbove(t *testing.T) {
	images := runContainersRealtimeWithThreshold(t, "high").Images
	assert.NotEmpty(t, images)
	for _, img := range images {
		assert.Len(t, img.Vulnerabilities, 1, "image %s", img.ImageName)
		assert.Equal(t, "CVE-H", img.Vulnerabilities[0].CVE)
	}
}

func TestContainersRealtime_SeverityThreshold_Low_KeepsLowAndAbove(t *testing.T) {
	images := runContainersRealtimeWithThreshold(t, "low").Images
	assert.NotEmpty(t, images)
	for _, img := range images {
		assert.Len(t, img.Vulnerabilities, 2, "image %s", img.ImageName)
	}
}

func TestContainersRealtime_SeverityThreshold_Critical_DropsImagesWithLowerStatus(t *testing.T) {
	assert.NotEmpty(t, runContainersRealtimeWithThreshold(t, "").Images)
	assert.Empty(t, runContainersRealtimeWithThreshold(t, "critical").Images, "images with status High must be dropped")
}

func secretsRealtimeResults(t *testing.T, extraArgs ...string) []secretsrealtime.SecretsRealtimeResult {
	enableRealtimeFlag()
	args := append([]string{"scan", "secrets-realtime", "-s", "data/secret-exposed.txt"}, extraArgs...)
	out, err := executeRedirectedTestCommand(args...)
	assert.NoError(t, err)

	var results []secretsrealtime.SecretsRealtimeResult
	assert.NoError(t, json.Unmarshal(out.Bytes(), &results))
	return results
}

func TestSecretsRealtime_SeverityThreshold_ValidValue_Success(t *testing.T) {
	enableRealtimeFlag()
	execCmdNilAssertion(t, "scan", "secrets-realtime", "-s", "data/secret-exposed.txt", "--severity-threshold", "high")
}

func TestSecretsRealtime_SeverityThreshold_InvalidValue_Fails(t *testing.T) {
	enableRealtimeFlag()
	err := execCmdNotNilAssertion(t, "scan", "secrets-realtime", "-s", "data/secret-exposed.txt", "--severity-threshold", invalidSeverityThresholdList)
	assert.Contains(t, err.Error(), invalidSeverityThresholdMsg)
}

func TestSecretsRealtime_SeverityThreshold_Info_ReturnsEverything(t *testing.T) {
	all := secretsRealtimeResults(t)
	assert.NotEmpty(t, all, "the fixture contains a secret")
	assert.Len(t, secretsRealtimeResults(t, "--severity-threshold", "info"), len(all))
}

func TestSecretsRealtime_SeverityThreshold_Critical_ReturnsOnlyCritical(t *testing.T) {
	for _, r := range secretsRealtimeResults(t, "--severity-threshold", "critical") {
		assert.Equal(t, "Critical", r.Severity, "secret %s", r.Title)
	}
}

func writeTerraformFile(t *testing.T) string {
	testFile := filepath.Join(t.TempDir(), "test.tf")
	assert.NoError(t, os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o600))
	return testFile
}

func TestIacRealtime_SeverityThreshold_InvalidValue_FailsBeforeEngineStarts(t *testing.T) {
	enableRealtimeFlag()
	err := execCmdNotNilAssertion(t, "scan", "iac-realtime", "-s", writeTerraformFile(t), "--severity-threshold", invalidSeverityThresholdList)
	assert.Contains(t, err.Error(), invalidSeverityThresholdMsg)
}

func TestIacRealtime_SeverityThreshold_ValidValue_PassesValidation(t *testing.T) {
	clearFlags() // realtime stays disabled, so the command stops at the tenant check instead of starting a container
	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.Flags().String(commonParams.SourcesFlag, writeTerraformFile(t), "")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "")
	cmd.Flags().String(commonParams.EngineFlag, "docker", "")
	cmd.Flags().String(commonParams.SeverityThresholdFlag, "high", "")

	err := RunScanIacRealtimeCommand(&mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{})(cmd, nil)
	assert.ErrorContains(t, err, errorconstants.RealtimeEngineNotAvailable)
}
