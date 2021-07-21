package commands

import (
	"github.com/checkmarxDev/ast-cli/internal/wrappers/mock"
	"log"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func TestMain(m *testing.M) {
	log.Println("Commands tests started")
	// Run all tests
	exitVal := m.Run()
	log.Println("Commands tests done")
	os.Exit(exitVal)
}

func createASTTestCommand() *cobra.Command {
	scansMockWrapper := &mock.ScansMockWrapper{}
	uploadsMockWrapper := &mock.UploadsMockWrapper{}
	projectsMockWrapper := &mock.ProjectsMockWrapper{}
	resultsMockWrapper := &mock.ResultsMockWrapper{}
	authWrapper := &mock.AuthMockWrapper{}
	return NewAstCLI(scansMockWrapper,
		uploadsMockWrapper,
		projectsMockWrapper,
		resultsMockWrapper,
		authWrapper,
	)
}

func TestRootHelp(t *testing.T) {
	cmd := createASTTestCommand()
	args := "--help"
	err := executeTestCommand(cmd, args)
	assert.NilError(t, err)
}

func TestRootVersion(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "version")
	assert.NilError(t, err)
}

func executeTestCommand(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	cmd.SilenceUsage = false
	return cmd.Execute()
}
