//go:build !integration

package commands

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
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
	aiOn := []string{allowAiUsageKey, "true"}
	llmOn := []string{sastExtendedAnalysisKey, "true"}
	llmOff := []string{sastExtendedAnalysisKey, "false"}
	tests := []struct {
		name          string
		tenant        []string
		project       []string
		projectID     string
		expected      bool
		projectCalled bool
	}{
		{name: "AI usage off: tenant and project LLM on ignored", tenant: slices.Concat([]string{allowAiUsageKey, "false"}, llmOn),
			project: llmOn, projectID: "p1", expected: false},
		{name: "AI usage missing is off", tenant: llmOn, project: llmOn, projectID: "p1", expected: false},
		{name: "tenant on, project on", tenant: slices.Concat(aiOn, llmOn), project: llmOn, projectID: "p1", expected: true, projectCalled: true},
		{name: "tenant on, project off", tenant: slices.Concat(aiOn, llmOn), project: llmOff, projectID: "p1", expected: false, projectCalled: true},
		{name: "tenant off, project on", tenant: slices.Concat(aiOn, llmOff), project: llmOn, projectID: "p1", expected: true, projectCalled: true},
		{name: "tenant off, project off", tenant: slices.Concat(aiOn, llmOff), project: llmOff, projectID: "p1", expected: false, projectCalled: true},
		{name: "tenant on, project not set", tenant: slices.Concat(aiOn, llmOn), projectID: "p1", expected: true, projectCalled: true},
		{name: "tenant off, project not set", tenant: aiOn, projectID: "p1", expected: false, projectCalled: true},
		{name: "tenant on, no project", tenant: slices.Concat(aiOn, llmOn), project: llmOff, projectID: "", expected: true},
		{name: "values are case-insensitive", tenant: []string{allowAiUsageKey, "TRUE"}, project: []string{sastExtendedAnalysisKey, " True "},
			projectID: "p1", expected: true, projectCalled: true},
		{name: "findings analysis is not checked", tenant: aiOn,
			project: []string{"scan.config.sast.findingsAnalysis", "false", sastExtendedAnalysisKey, "true"}, projectID: "p1", expected: true, projectCalled: true},
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

func TestIsLlmScanningEnabled_SettingsErrors(t *testing.T) {
	failingTenant := mock.TenantConfigurationMockWrapper{
		CustomGetTenantConfiguration: func() (*[]*wrappers.TenantConfigurationResponse, *wrappers.WebError, error) {
			return nil, nil, errors.New("tenant unavailable")
		},
	}
	tenantWebError := mock.TenantConfigurationMockWrapper{
		CustomGetTenantConfiguration: func() (*[]*wrappers.TenantConfigurationResponse, *wrappers.WebError, error) {
			return nil, &wrappers.WebError{Message: "bad request"}, nil
		},
	}
	// Allow AI Usage cannot be confirmed, so the project is not read and the filter stays as it is.
	called := false
	assert.Assert(t, !isLlmScanningEnabled(failingTenant, projectSettings(&called, sastExtendedAnalysisKey, "true"), "p1"))
	assert.Assert(t, !isLlmScanningEnabled(tenantWebError, projectSettings(&called, sastExtendedAnalysisKey, "true"), "p1"))
	assert.Assert(t, !called, "project settings must not be read when the tenant call fails")

	// A failed project call uses the tenant setting.
	failingProject := projectSettings(&called, sastExtendedAnalysisKey, "false")
	failingProject.err = errors.New("project unavailable")
	assert.Assert(t, isLlmScanningEnabled(tenantSettings(allowAiUsageKey, "true", sastExtendedAnalysisKey, "true"), failingProject, "p1"))
	assert.Assert(t, !isLlmScanningEnabled(tenantSettings(allowAiUsageKey, "true", sastExtendedAnalysisKey, "false"), failingProject, "p1"))
}

func TestResolveDefaultIncludeFilters(t *testing.T) {
	called := false
	on := tenantSettings(allowAiUsageKey, "true", sastExtendedAnalysisKey, "true")
	assert.DeepEqual(t, resolveDefaultIncludeFilters(on, projectSettings(&called), "p1"), commonParams.BaseIncludeFilters)

	off := tenantSettings(allowAiUsageKey, "false", sastExtendedAnalysisKey, "true")
	filters := resolveDefaultIncludeFilters(off, projectSettings(&called), "p1")
	assert.DeepEqual(t, filters, commonParams.SastIncludeFilters)
	assert.Assert(t, !slices.Contains(filters, "*.sh"), "Linguist patterns must be left out when LLM-based scanning is off")
	assert.Assert(t, slices.Contains(filters, "*.java"), "SAST patterns must stay when LLM-based scanning is off")
}

func writeSourceFiles(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, rel := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		assert.NilError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		assert.NilError(t, os.WriteFile(p, []byte("content of "+rel), 0o600))
	}
	return root + string(os.PathSeparator)
}

func compressedEntries(t *testing.T, root string, defaultIncludeFilters []string) []string {
	t.Helper()
	zipPath, err := compressFolder(root, "", "", "", filtering.NopMatcher{}, false, false, false, defaultIncludeFilters)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = os.Remove(zipPath) })
	data, err := os.ReadFile(zipPath)
	assert.NilError(t, err)
	return zipEntryNames(t, data)
}

func TestCompressFolder_LinguistFilesFollowLlmSetting(t *testing.T) {
	root := writeSourceFiles(t, "Main.java", "deploy.sh", "Jenkinsfile")

	withLinguist := compressedEntries(t, root, commonParams.DefaultIncludeFilters(true))
	assert.DeepEqual(t, []string{"Jenkinsfile", "Main.java", "deploy.sh"}, withLinguist)

	withoutLinguist := compressedEntries(t, root, commonParams.DefaultIncludeFilters(false))
	assert.DeepEqual(t, []string{"Main.java"}, withoutLinguist)
}

// filterFlagsCommand builds a command with the flags getScanDefaultIncludeFilters reads.
func filterFlagsCommand(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, "", "")
	cmd.Flags().Bool(commonParams.SkipDefaultFilterFlag, false, "")
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
		{name: "LLM settings decide for a local source", args: []string{"--" + commonParams.SourcesFlag, "."},
			expected: commonParams.BaseIncludeFilters, tenantCalled: true},
		{name: "--skip-default-filter uses no default filter", args: []string{"--" + commonParams.SourcesFlag, ".", "--" + commonParams.SkipDefaultFilterFlag},
			expected: nil, tenantCalled: false},
		{name: "git source builds no zip", args: []string{"--" + commonParams.SourcesFlag, "https://github.com/org/repo.git"},
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
						{Key: sastExtendedAnalysisKey, Value: "true"},
					}
					return &config, nil, nil
				},
			}
			got := getScanDefaultIncludeFilters(filterFlagsCommand(t, tt.args...), llmOn, projectSettings(&projectCalled), "p1")
			assert.DeepEqual(t, tt.expected, got)
			assert.Equal(t, tt.tenantCalled, tenantCalled, "tenant configuration call")
			assert.Equal(t, tt.tenantCalled, projectCalled, "the project is read first; it does not set the value here, so the tenant decides")
		})
	}
}
