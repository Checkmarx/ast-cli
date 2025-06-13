package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestPreReceiveCommand(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{}
	cmd := PreReceiveCommand(mockJWT)
	assert.NotNil(t, cmd)
	assert.Equal(t, "pre-receive", cmd.Use)
	subCmds := cmd.Commands()
	subCmdName := make([]string, len(subCmds))
	for i, subCmd := range subCmds {
		subCmdName[i] = subCmd.Name()
	}
	expectedSubCmds := []string{
		"secrets-scan",
	}

	for i, expectedSubCmd := range expectedSubCmds {
		assert.Contains(t, expectedSubCmd, subCmdName[i])
	}
}

func TestPreReceiveCommand_withConfig(t *testing.T) {
	cmd := createASTTestCommand()
	workDir, _ := os.Getwd()
	configFile := filepath.Join(workDir, "config.yaml")
	_ = os.WriteFile(configFile, []byte(""), 0644)
	err := executeTestCommand(
		cmd,
		"hooks", "pre-receive", "secrets-scan", "--config", "config.yaml",
	)
	assert.Nil(t, err)
}

func TestPreReceiveCommand_withWrongFlagConfig(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"hooks", "pre-receive", "secrets-scan", "--cf", "/path/config.yaml",
	)
	assert.NotNil(t, err)
}
