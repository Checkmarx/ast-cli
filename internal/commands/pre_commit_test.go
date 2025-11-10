package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewHooksCommand(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{}
	mockFF := &mock.FeatureFlagsMockWrapper{}
	cmd := NewHooksCommand(mockJWT, mockFF)

	assert.NotNil(t, cmd)
	assert.Equal(t, "hooks", cmd.Use)
	assert.Contains(t, cmd.Short, "Manage Git hooks")
}

func TestPreCommitCommand(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{}
	mockFF := &mock.FeatureFlagsMockWrapper{}
	cmd := PreCommitCommand(mockJWT, mockFF)

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
		name                   string
		scsLicensingV2         bool
		allowSecretDetection   int
		allowEnterpriseSecrets int
		aiEnabled              int
		wantErr                bool
	}{
		{
			name:                   "scsLicensing V2 ON and Secret Detection allowed",
			scsLicensingV2:         true,
			allowSecretDetection:   0,
			allowEnterpriseSecrets: mock.EnterpriseSecretsDisabled,
			wantErr:                false,
		},
		{
			name:                   "scsLicensing V2 ON and Secret Detection denied",
			scsLicensingV2:         true,
			allowSecretDetection:   mock.SecretDetectionDisabled,
			allowEnterpriseSecrets: 0,
			wantErr:                true,
		},
		{
			name:                   "scsLicensing V2 OFF and Enterprise Secrets allowed",
			scsLicensingV2:         false,
			allowSecretDetection:   mock.SecretDetectionDisabled,
			allowEnterpriseSecrets: 0,
			wantErr:                false,
		},
		{
			name:                   "scsLicensing V2 OFF and Enterprise Secrets denied",
			scsLicensingV2:         false,
			allowSecretDetection:   0,
			allowEnterpriseSecrets: mock.EnterpriseSecretsDisabled,
			wantErr:                true,
		},
		{
			name:                   "AI enabled and secrets license disabled",
			aiEnabled:              1,
			allowEnterpriseSecrets: mock.EnterpriseSecretsDisabled,
			allowSecretDetection:   mock.SecretDetectionDisabled,
			wantErr:                true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappers.ClearCache()
			mockFF := &mock.FeatureFlagsMockWrapper{}
			mock.Flag = wrappers.FeatureFlagResponseModel{
				Name:   wrappers.ScsLicensingV2Enabled,
				Status: tt.scsLicensingV2,
			}

			mockJWT := &mock.JWTMockWrapper{}
			mockJWT.AIEnabled = tt.aiEnabled
			mockJWT.EnterpriseSecretsEnabled = tt.allowEnterpriseSecrets
			mockJWT.SecretDetectionEnabled = tt.allowSecretDetection

			err := validateLicense(mockJWT, mockFF)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "License validation failed")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecretsIgnoreCommand(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{}
	mockFF := &mock.FeatureFlagsMockWrapper{}
	cmd := secretsIgnoreCommand(mockJWT, mockFF)
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
