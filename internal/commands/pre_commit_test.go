package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockJWTWrapper struct {
	allowedEngine bool
	returnError   error
}

func (m *mockJWTWrapper) IsAllowedEngine(engine string) (bool, error) {
	return m.allowedEngine, m.returnError
}

func (m *mockJWTWrapper) GetAllowedEngines(featureFlagsWrapper wrappers.FeatureFlagsWrapper) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (m *mockJWTWrapper) ExtractTenantFromToken() (string, error) {
	return "test-tenant", nil
}

type mockFeatureFlagsWrapper struct{}

func (m *mockFeatureFlagsWrapper) GetAll() (*wrappers.FeatureFlagsResponseModel, error) {
	return &wrappers.FeatureFlagsResponseModel{}, nil
}

func (m *mockFeatureFlagsWrapper) GetSpecificFlag(specificFlag string) (*wrappers.FeatureFlagResponseModel, error) {
	return &wrappers.FeatureFlagResponseModel{
		Name:   specificFlag,
		Status: true,
	}, nil
}

func TestNewHooksCommand(t *testing.T) {
	mockJWT := &mockJWTWrapper{}
	mockFeatureFlags := &mockFeatureFlagsWrapper{}

	cmd := NewHooksCommand(mockJWT, mockFeatureFlags)

	assert.NotNil(t, cmd)
	assert.Equal(t, "hooks", cmd.Use)
	assert.Contains(t, cmd.Short, "Manage Git hooks")
}

func TestPreCommitCommand(t *testing.T) {
	mockJWT := &mockJWTWrapper{}
	mockFeatureFlags := &mockFeatureFlagsWrapper{}

	cmd := PreCommitCommand(mockJWT, mockFeatureFlags)

	assert.NotNil(t, cmd)
	assert.Equal(t, "pre-commit", cmd.Use)

	subCmds := cmd.Commands()
	subCmdNames := make([]string, len(subCmds))
	for i, subCmd := range subCmds {
		subCmdNames[i] = subCmd.Name()
	}

	expectedSubCommands := []string{
		"secrets-install-git-hook",
		"secrets-uninstall-git-hook",
		"secrets-update-git-hook",
		"secrets-scan",
		"secrets-ignore",
		"secrets-help",
	}

	for _, expectedCmd := range expectedSubCommands {
		assert.Contains(t, subCmdNames, expectedCmd)
	}
}

func TestValidateLicense(t *testing.T) {
	tests := []struct {
		name        string
		allowed     bool
		returnError error
		wantError   bool
	}{
		{"License is valid", true, nil, false},
		{"License is invalid", false, nil, true},
		{"Error checking license", false, errors.New("failed to check license"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockJWT := &mockJWTWrapper{
				allowedEngine: tt.allowed,
				returnError:   tt.returnError,
			}

			err := validateLicense(mockJWT)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecretsIgnoreCommand(t *testing.T) {
	mockJWT := &mockJWTWrapper{
		allowedEngine: true,
		returnError:   nil,
	}

	cmd := secretsIgnoreCommand(mockJWT)
	assert.NotNil(t, cmd)

	resultIdsFlag := cmd.Flag("resultIds")
	assert.NotNil(t, resultIdsFlag)
	assert.Equal(t, "resultIds", resultIdsFlag.Name)

	allFlag := cmd.Flag("all")
	assert.NotNil(t, allFlag)
	assert.Equal(t, "all", allFlag.Name)

	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)

	cmd.Flags().Set("resultIds", "id1,id2")
	cmd.Flags().Set("all", "true")
	err = cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used with")

	cmd.Flags().Set("resultIds", "")
	cmd.Flags().Set("all", "false")
	err = cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be specified")
}

func TestSecretsHelpCommand(t *testing.T) {
	cmd := secretsHelpCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "secrets-help", cmd.Use)
	assert.Contains(t, cmd.Short, "Display help")
}
