// +build !integration

package commands

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"

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
	scansMockWrapper := &wrappers.ScansMockWrapper{}
	uploadsMockWrapper := &wrappers.UploadsMockWrapper{}
	projectsMockWrapper := &wrappers.ProjectsMockWrapper{}
	resultsMockWrapper := &wrappers.ResultsMockWrapper{}
	bflMockWrapper := &wrappers.BFLMockWrapper{}
	return NewAstCLI(scansMockWrapper, uploadsMockWrapper, projectsMockWrapper, resultsMockWrapper, bflMockWrapper)
}

func TestRootHelp(t *testing.T) {
	cmd := createASTTestCommand()
	args := fmt.Sprintf("--help")
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
	cmd.SilenceUsage = true
	return cmd.Execute()
}
