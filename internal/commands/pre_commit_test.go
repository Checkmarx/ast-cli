package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewHooksCommand(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{}
	cmd := NewHooksCommand(mockJWT)

	assert.NotNil(t, cmd)
	assert.Equal(t, "hooks", cmd.Use)
	assert.Contains(t, cmd.Short, "Manage Git hooks")
}

func TestPreCommitCommand(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{}
	cmd := PreCommitCommand(mockJWT)

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
		name      string
		aiEnabled int
		wantError bool
	}{
		{
			name:      "License is valid",
			aiEnabled: 0,
			wantError: false,
		},
		{
			name:      "License is invalid",
			aiEnabled: mock.AIProtectionDisabled,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockJWT := &mock.JWTMockWrapper{
				AIEnabled: tt.aiEnabled,
			}

			err := ValidateLicense(mockJWT)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecretsIgnoreCommand(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{}
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
