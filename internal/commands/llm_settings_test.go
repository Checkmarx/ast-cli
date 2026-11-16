//go:build !integration

package commands

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/filtering"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func tenantSettings(kv ...string) mock.TenantConfigurationMockWrapper {
	var config []*wrappers.TenantConfigurationResponse
	for i := 0; i+1 < len(kv); i += 2 {
		config = append(config, &wrappers.TenantConfigurationResponse{Key: kv[i], Value: kv[i+1]})
	}
	return mock.TenantConfigurationMockWrapper{
		CustomGetTenantConfiguration: func() (*[]*wrappers.TenantConfigurationResponse, *wrappers.WebError, error) {
			return &config, nil, nil
		},
	}
}

// projectSettingsWrapper serves a fixed project configuration and records whether it was asked for it.
type projectSettingsWrapper struct {
	*mock.ProjectsMockWrapper
	config []wrappers.ProjectConfiguration
	err    error
	called *bool
}

func (p projectSettingsWrapper) GetConfiguration(string) ([]wrappers.ProjectConfiguration, *wrappers.ErrorModel, error) {
	*p.called = true
	return p.config, nil, p.err
}

func projectSettings(called *bool, kv ...string) projectSettingsWrapper {
	var config []wrappers.ProjectConfiguration
	for i := 0; i+1 < len(kv); i += 2 {
		config = append(config, wrappers.ProjectConfiguration{Key: kv[i], Value: kv[i+1]})
	}
	return projectSettingsWrapper{ProjectsMockWrapper: &mock.ProjectsMockWrapper{}, config: config, called: called}
}

func TestIsLlmScanningEnabled(t *testing.T) {
	llmOn := []string{sastFindingsAnalysisKey, "true", sastExtendedAnalysisKey, "true"}
	tests := []struct {
		name          string
		tenant        []string
		project       []string
		projectID     string
		expected      bool
		projectCalled bool
	}{
		{name: "AI usage off ignores LLM settings", tenant: append([]string{allowAiUsageKey, "false"}, llmOn...), project: llmOn,
			projectID: "p1", expected: false, projectCalled: false},
		{name: "AI usage missing is off", tenant: llmOn, project: llmOn, projectID: "p1", expected: false, projectCalled: false},
		{name: "tenant LLM on", tenant: append([]string{allowAiUsageKey, "true"}, llmOn...), projectID: "p1", expected: true, projectCalled: false},
		{name: "values are case-insensitive", tenant: []string{allowAiUsageKey, "TRUE", sastFindingsAnalysisKey, "True", sastExtendedAnalysisKey, " true "},
			projectID: "p1", expected: true, projectCalled: false},
		{name: "tenant needs both LLM keys, project on", tenant: []string{allowAiUsageKey, "true", sastExtendedAnalysisKey, "true"},
			project: llmOn, projectID: "p1", expected: true, projectCalled: true},
		{name: "tenant off, project on", tenant: []string{allowAiUsageKey, "true", sastFindingsAnalysisKey, "false", sastExtendedAnalysisKey, "false"},
			project: llmOn, projectID: "p1", expected: true, projectCalled: true},
		{name: "tenant off, project needs both keys", tenant: []string{allowAiUsageKey, "true"},
			project: []string{sastFindingsAnalysisKey, "true", sastExtendedAnalysisKey, "false"}, projectID: "p1", expected: false, projectCalled: true},
		{name: "tenant off, no project", tenant: []string{allowAiUsageKey, "true"}, project: llmOn, projectID: "", expected: false, projectCalled: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			got := isLlmScanningEnabled(tenantSettings(tt.tenant...), projectSettings(&called, tt.project...), tt.projectID)
			assert.Equal(t, tt.expected, got)
			assert.Equal(t, tt.projectCalled, called, "project configuration call")
		})
	}
}

func TestIsLlmScanningEnabled_SettingsErrorsDisableLinguist(t *testing.T) {
	called := false
	failingTenant := mock.TenantConfigurationMockWrapper{
		CustomGetTenantConfiguration: func() (*[]*wrappers.TenantConfigurationResponse, *wrappers.WebError, error) {
			return nil, nil, errors.New("tenant unavailable")
		},
	}
	assert.Assert(t, !isLlmScanningEnabled(failingTenant, projectSettings(&called), "p1"))
	assert.Assert(t, !called, "project settings must not be read when the tenant call fails")

	tenantWebError := mock.TenantConfigurationMockWrapper{
		CustomGetTenantConfiguration: func() (*[]*wrappers.TenantConfigurationResponse, *wrappers.WebError, error) {
			return nil, &wrappers.WebError{Message: "bad request"}, nil
		},
	}
	assert.Assert(t, !isLlmScanningEnabled(tenantWebError, projectSettings(&called), "p1"))

	failingProject := projectSettings(&called)
	failingProject.err = errors.New("project unavailable")
	assert.Assert(t, !isLlmScanningEnabled(tenantSettings(allowAiUsageKey, "true"), failingProject, "p1"))
	assert.Assert(t, called)
}

func TestResolveDefaultIncludeFilters(t *testing.T) {
	called := false
	on := tenantSettings(allowAiUsageKey, "true", sastFindingsAnalysisKey, "true", sastExtendedAnalysisKey, "true")
	assert.DeepEqual(t, resolveDefaultIncludeFilters(on, projectSettings(&called), "p1"), commonParams.BaseIncludeFilters)

	off := tenantSettings(allowAiUsageKey, "false")
	filters := resolveDefaultIncludeFilters(off, projectSettings(&called), "p1")
	assert.DeepEqual(t, filters, commonParams.SastIncludeFilters)
	assert.Assert(t, !slices.Contains(filters, "*.sh"), "Linguist patterns must be left out when LLM-based scanning is off")
	assert.Assert(t, slices.Contains(filters, "*.java"), "SAST patterns must stay when LLM-based scanning is off")
}

func withDefaultFilterSizeLimit(t *testing.T, limit int64) {
	t.Helper()
	previous := MaxDefaultFilterFileSizeBytes
	MaxDefaultFilterFileSizeBytes = limit
	t.Cleanup(func() { MaxDefaultFilterFileSizeBytes = previous })
}

func writeSizedFiles(t *testing.T, files map[string]int) string {
	t.Helper()
	root := t.TempDir()
	for rel, size := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		assert.NilError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		assert.NilError(t, os.WriteFile(p, bytes.Repeat([]byte("a"), size), 0o600))
	}
	return root + string(os.PathSeparator)
}

func compressedEntries(t *testing.T, root, userIncludeFilter string, skipDefaultFilter bool, defaultIncludeFilters []string) []string {
	t.Helper()
	zipPath, err := compressFolder(root, "", userIncludeFilter, "", filtering.NopMatcher{}, skipDefaultFilter, false, false, defaultIncludeFilters)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = os.Remove(zipPath) })
	data, err := os.ReadFile(zipPath)
	assert.NilError(t, err)
	return zipEntryNames(t, data)
}

func TestDefaultFilterSizeLimitIs100MB(t *testing.T) {
	assert.Equal(t, int64(100*1024*1024), MaxDefaultFilterFileSizeBytes)
}

func TestExceedsDefaultFilterSizeLimit(t *testing.T) {
	withDefaultFilterSizeLimit(t, 10)
	tests := []struct {
		name     string
		filters  []string
		file     string
		size     int64
		expected bool
	}{
		{name: "default-filter file over the limit", filters: commonParams.BaseIncludeFilters, file: "Main.java", size: 11, expected: true},
		{name: "file exactly at the limit is kept", filters: commonParams.BaseIncludeFilters, file: "Main.java", size: 10, expected: false},
		{name: "SCA file over the limit is kept", filters: commonParams.BaseIncludeFilters, file: "lib.JAR", size: 11, expected: false},
		{name: "SCA results file is kept", filters: commonParams.BaseIncludeFilters, file: ".cxsca-results.json", size: 11, expected: false},
		{name: "file outside the default filter is not limited", filters: commonParams.BaseIncludeFilters, file: "big.foo", size: 11, expected: false},
		{name: "no default filter means no limit", filters: nil, file: "Main.java", size: 11, expected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, exceedsDefaultFilterSizeLimit(tt.filters, tt.file, tt.size))
		})
	}
}

func TestCompressFolder_SizeLimitOnDefaultFilterFiles(t *testing.T) {
	withDefaultFilterSizeLimit(t, 10)
	root := writeSizedFiles(t, map[string]int{
		"small.java": 5, "big.java": 11, "src/big.py": 50, "lib/big.jar": 50, "custom.foo": 50,
	})
	got := compressedEntries(t, root, "*.foo", false, commonParams.BaseIncludeFilters)
	assert.DeepEqual(t, []string{"custom.foo", "lib/big.jar", "small.java"}, got)
}

func TestCompressFolder_LinguistFilesFollowLlmSetting(t *testing.T) {
	root := writeSizedFiles(t, map[string]int{"Main.java": 5, "deploy.sh": 5, "Jenkinsfile": 5})

	withLinguist := compressedEntries(t, root, "", false, commonParams.DefaultIncludeFilters(true))
	assert.DeepEqual(t, []string{"Jenkinsfile", "Main.java", "deploy.sh"}, withLinguist)

	withoutLinguist := compressedEntries(t, root, "", false, commonParams.DefaultIncludeFilters(false))
	assert.DeepEqual(t, []string{"Main.java"}, withoutLinguist)
}

func TestCompressFolder_SizeLimitSkippedWithSkipDefaultFilter(t *testing.T) {
	withDefaultFilterSizeLimit(t, 10)
	root := writeSizedFiles(t, map[string]int{"big.java": 50})
	got := compressedEntries(t, root, "", true, commonParams.BaseIncludeFilters)
	assert.DeepEqual(t, []string{"big.java"}, got)
}

func TestCompressFolder_OnlyOversizedFilesAddsPlaceholder(t *testing.T) {
	withDefaultFilterSizeLimit(t, 10)
	root := writeSizedFiles(t, map[string]int{"big.java": 50, "src/big.go": 50})
	got := compressedEntries(t, root, "", false, commonParams.BaseIncludeFilters)
	assert.DeepEqual(t, []string{".container-scan"}, got)
}

// captureStdLog sends the standard logger (used by logger.Print) to w until the returned func is called.
func captureStdLog(w io.Writer) func() {
	log.SetOutput(w)
	return func() { log.SetOutput(os.Stderr) }
}

func TestHandleFile_LogsOversizedFile(t *testing.T) {
	withDefaultFilterSizeLimit(t, 10)
	root := writeSizedFiles(t, map[string]int{"big.java": 50})
	var logs bytes.Buffer
	restore := captureStdLog(&logs)
	defer restore()

	zw := zip.NewWriter(&bytes.Buffer{})
	err := addDirFiles(zw, "", root, commonParams.BaseExcludeFilters, commonParams.BaseIncludeFilters, commonParams.BaseIncludeFilters,
		filtering.NopMatcher{}, false)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(logs.String(), "Excluded (larger than 0MB)") && strings.Contains(logs.String(), "big.java"), logs.String())
}

// filterFlagsCommand builds a command with the flags getScanDefaultIncludeFilters reads.
func filterFlagsCommand(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, "", "")
	cmd.Flags().Bool(commonParams.SkipDefaultFilterFlag, false, "")
	cmd.Flags().Bool(commonParams.DefaultFilterFlag, false, "")
	assert.NilError(t, cmd.Flags().Parse(args))
	return cmd
}

func TestGetScanDefaultIncludeFilters(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expected     []string
		tenantCalled bool
	}{
		{name: "LLM settings decide by default", args: []string{"--" + commonParams.SourcesFlag, "."},
			expected: commonParams.BaseIncludeFilters, tenantCalled: true},
		{name: "--default-filter keeps SAST patterns only", args: []string{"--" + commonParams.SourcesFlag, ".", "--" + commonParams.DefaultFilterFlag},
			expected: commonParams.SastIncludeFilters, tenantCalled: false},
		{name: "--skip-default-filter uses no default filter", args: []string{"--" + commonParams.SourcesFlag, ".", "--" + commonParams.SkipDefaultFilterFlag},
			expected: nil, tenantCalled: false},
		{name: "git source builds no zip", args: []string{"--" + commonParams.SourcesFlag, "https://github.com/org/repo.git", "--" + commonParams.DefaultFilterFlag},
			expected: nil, tenantCalled: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantCalled, projectCalled := false, false
			llmOn := mock.TenantConfigurationMockWrapper{
				CustomGetTenantConfiguration: func() (*[]*wrappers.TenantConfigurationResponse, *wrappers.WebError, error) {
					tenantCalled = true
					config := []*wrappers.TenantConfigurationResponse{
						{Key: allowAiUsageKey, Value: "true"},
						{Key: sastFindingsAnalysisKey, Value: "true"},
						{Key: sastExtendedAnalysisKey, Value: "true"},
					}
					return &config, nil, nil
				},
			}
			got := getScanDefaultIncludeFilters(filterFlagsCommand(t, tt.args...), llmOn, projectSettings(&projectCalled), "p1")
			assert.DeepEqual(t, tt.expected, got)
			assert.Equal(t, tt.tenantCalled, tenantCalled, "tenant configuration call")
			assert.Assert(t, !projectCalled, "tenant settings decide here, so the project must not be read")
		})
	}
}

func TestCompressFolder_DefaultFilterFlagLeavesOutLinguistFiles(t *testing.T) {
	root := writeSizedFiles(t, map[string]int{"Main.java": 5, "deploy.sh": 5, "Jenkinsfile": 5})
	cmd := filterFlagsCommand(t, "--"+commonParams.SourcesFlag, root, "--"+commonParams.DefaultFilterFlag)
	filters := getScanDefaultIncludeFilters(cmd, tenantSettings(), projectSettings(new(bool)), "p1")
	assert.DeepEqual(t, []string{"Main.java"}, compressedEntries(t, root, "", false, filters))
}

func TestCreateScanDefaultFilter_Wiring(t *testing.T) {
	execCmdNilAssertion(t,
		"scan", "create", "--project-name", "MOCK", "-s", "data", "-b", "dummy_branch",
		"--default-filter",
	)
}

func TestCreateScanDefaultFilterWithSkipDefaultFilter_Fails(t *testing.T) {
	err := execCmdNotNilAssertion(t,
		"scan", "create", "--project-name", "MOCK", "-s", "data", "-b", "dummy_branch",
		"--default-filter", "--skip-default-filter",
	)
	assert.ErrorContains(t, err, "--default-filter and --skip-default-filter cannot be used together")
}
