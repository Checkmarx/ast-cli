//go:build !integration

package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
	invalidSeverityFilterValue = "critical,hgh"
	invalidSeverityFilterMsg   = "invalid severity-filter value(s): hgh"
	acceptedSeverityValuesMsg  = "critical, high, medium, low, info"
)

func enableRealtimeFlag() {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
}

func TestOssRealtime_SeverityFilter_ValidValue_Success(t *testing.T) {
	enableRealtimeFlag()
	execCmdNilAssertion(t,
		"scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-filter", "Critical, High",
	)
}

func TestOssRealtime_SeverityFilter_EmptyValue_TreatedAsNoFilter(t *testing.T) {
	enableRealtimeFlag()
	execCmdNilAssertion(t,
		"scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-filter", "",
	)
}

func TestOssRealtime_SeverityFilter_InvalidValue_Fails(t *testing.T) {
	enableRealtimeFlag()
	err := execCmdNotNilAssertion(t,
		"scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-filter", invalidSeverityFilterValue,
	)
	assert.Contains(t, err.Error(), invalidSeverityFilterMsg)
	assert.Contains(t, err.Error(), acceptedSeverityValuesMsg)
}

func TestOssRealtime_SeverityFilter_OutputOnlyContainsAllowedSeverities(t *testing.T) {
	enableRealtimeFlag()

	rt := &mock.RealtimeScannerMockWrapper{
		CustomScanPackages: func(req *wrappers.RealtimeScannerPackageRequest) (*wrappers.RealtimeScannerPackageResponse, error) {
			var resp wrappers.RealtimeScannerPackageResponse
			for _, p := range req.Packages {
				resp.Packages = append(resp.Packages, wrappers.RealtimeScannerResults{
					PackageManager: p.PackageManager,
					PackageName:    p.PackageName,
					Version:        p.Version,
					Status:         "OK",
					Vulnerabilities: []wrappers.RealtimeScannerVulnerability{
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
	cmd.Flags().String(commonParams.SourcesFlag, "data/manifests/package.json", "")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "")
	cmd.Flags().String(commonParams.SeverityFilterFlag, "high", "")

	err := RunScanOssRealtimeCommand(rt, &mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{})(cmd, nil)
	assert.NoError(t, err)

	var results ossrealtime.OssPackageResults
	assert.NoError(t, json.Unmarshal(out.Bytes(), &results))
	assert.NotEmpty(t, results.Packages)
	for _, p := range results.Packages {
		if p.Status == "Malicious" {
			continue
		}
		for _, v := range p.Vulnerabilities {
			assert.Equal(t, "High", v.Severity, "package %s returned a vulnerability outside the filter", p.PackageName)
		}
	}
}

func TestContainersRealtime_SeverityFilter_ValidValue_Success(t *testing.T) {
	enableRealtimeFlag()
	execCmdNilAssertion(t, "scan", "containers-realtime", "-s", "data/Dockerfile", "--severity-filter", "critical,high")
}

func TestContainersRealtime_SeverityFilter_InvalidValue_Fails(t *testing.T) {
	enableRealtimeFlag()
	err := execCmdNotNilAssertion(t, "scan", "containers-realtime", "-s", "data/Dockerfile", "--severity-filter", invalidSeverityFilterValue)
	assert.Contains(t, err.Error(), invalidSeverityFilterMsg)
}

func TestContainersRealtime_SeverityFilter_OutputOnlyContainsAllowedSeverities(t *testing.T) {
	enableRealtimeFlag()

	rt := &mock.RealtimeScannerMockWrapper{
		CustomScanImages: func(req *wrappers.ContainerImageRequest) (*wrappers.ContainerImageResponse, error) {
			var resp wrappers.ContainerImageResponse
			for _, img := range req.Images {
				resp.Images = append(resp.Images, wrappers.ContainerImageResponseItem{
					ImageName: img.ImageName,
					ImageTag:  img.ImageTag,
					Status:    "OK",
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
	cmd.Flags().String(commonParams.SeverityFilterFlag, "low", "")

	err := RunScanContainersRealtimeCommand(rt, &mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{})(cmd, nil)
	assert.NoError(t, err)

	var results containersrealtime.ContainerImageResults
	assert.NoError(t, json.Unmarshal(out.Bytes(), &results))
	assert.NotEmpty(t, results.Images)
	for _, img := range results.Images {
		if img.Status != "OK" {
			continue
		}
		assert.Len(t, img.Vulnerabilities, 1, "image %s should keep exactly the Low vulnerability", img.ImageName)
		for _, v := range img.Vulnerabilities {
			assert.Equal(t, "Low", v.Severity)
		}
	}
}

func TestSecretsRealtime_SeverityFilter_ValidValue_Success(t *testing.T) {
	enableRealtimeFlag()
	execCmdNilAssertion(t, "scan", "secrets-realtime", "-s", "data/secret-exposed.txt", "--severity-filter", "critical,high")
}

func TestSecretsRealtime_SeverityFilter_InvalidValue_Fails(t *testing.T) {
	enableRealtimeFlag()
	err := execCmdNotNilAssertion(t, "scan", "secrets-realtime", "-s", "data/secret-exposed.txt", "--severity-filter", invalidSeverityFilterValue)
	assert.Contains(t, err.Error(), invalidSeverityFilterMsg)
}

func TestSecretsRealtime_SeverityFilter_OutputOnlyContainsAllowedSeverities(t *testing.T) {
	enableRealtimeFlag()
	out, err := executeRedirectedTestCommand("scan", "secrets-realtime", "-s", "data/secret-exposed.txt", "--severity-filter", "critical")
	assert.NoError(t, err)

	var results []secretsrealtime.SecretsRealtimeResult
	assert.NoError(t, json.Unmarshal(out.Bytes(), &results))
	for _, r := range results {
		assert.Equal(t, "Critical", r.Severity, "secret %s returned outside the filter", r.Title)
	}
}

func TestIacRealtime_SeverityFilter_InvalidValue_FailsBeforeEngineStarts(t *testing.T) {
	enableRealtimeFlag()
	testFile := filepath.Join(t.TempDir(), "test.tf")
	assert.NoError(t, os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o600))

	err := execCmdNotNilAssertion(t, "scan", "iac-realtime", "-s", testFile, "--severity-filter", invalidSeverityFilterValue)
	assert.Contains(t, err.Error(), invalidSeverityFilterMsg)
}

func TestIacRealtime_SeverityFilter_ValidValue_FlagAccepted(t *testing.T) {
	enableRealtimeFlag()
	testFile := filepath.Join(t.TempDir(), "test.tf")
	assert.NoError(t, os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o600))

	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "")
	cmd.Flags().String(commonParams.EngineFlag, "docker", "")
	cmd.Flags().String(commonParams.SeverityFilterFlag, "critical,high", "")

	err := RunScanIacRealtimeCommand(&mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{})(cmd, nil)
	if err != nil {
		// Without a container engine an engine error is fine; only a severity-filter error fails the test.
		assert.NotContains(t, err.Error(), "severity-filter")
	}
}
