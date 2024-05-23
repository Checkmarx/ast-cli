package util

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"gotest.tools/assert"
)

const mockFormatErrorMessage = "Invalid format MOCK"

func TestNewUtilsCommand(t *testing.T) {
	cmd := NewUtilsCommand(mock.GitHubMockWrapper{},
		mock.AzureMockWrapper{},
		mock.BitBucketMockWrapper{},
		nil,
		mock.GitLabMockWrapper{},
		nil,
		mock.LearnMoreMockWrapper{},
		mock.TenantConfigurationMockWrapper{},
		mock.ChatMockWrapper{},
		nil,
		nil,
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.ByorMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	assert.Assert(t, cmd != nil, "Utils command must exist")
}

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		array  []string
		val    string
		exists bool
	}{
		{"Value exists in array", []string{"a", "b", "c"}, "b", true},
		{"Value does not exist in array", []string{"x", "y", "z"}, "a", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Contains(tt.array, tt.val); got != tt.exists {
				t.Errorf("Contains() = %v, want %v", got, tt.exists)
			}
		})
	}
}

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Valid Git URL", "https://github.com/username/repository.git", true},
		{"Invalid Git URL", "notagiturl", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGitURL(tt.url); got != tt.expected {
				t.Errorf("IsGitURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsSSHURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Valid SSH URL", "user@host:path/to/repository.git", true},
		{"Invalid SSH URL", "notasshurl", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSSHURL(tt.url); got != tt.expected {
				t.Errorf("IsSSHURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}
