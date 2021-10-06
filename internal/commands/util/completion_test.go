package util

import (
	"github.com/spf13/cobra"
	"gotest.tools/assert"
	"testing"
)

func TestNewCompletionCommand(t *testing.T) {
	cmd := NewCompletionCommand()
	assert.Assert(t, cmd != nil, "Completion command must exist")

	err := cmd.Execute()
	assert.Assert(t, err != nil, "Shell type must be defined")

	testShell(cmd, "bash", t)
	testShell(cmd, "zsh", t)
	testShell(cmd, "fish", t)
	testShell(cmd, "powershell", t)

	//TODO: catch console to check completion was printed out
	//TODO: wrong type does not return error
	testShell(cmd, "cenas", t)
}

func testShell(cmd *cobra.Command, shellType string, t *testing.T) {
	args := []string{"-s", shellType}
	cmd.SetArgs(args)
	err := cmd.Execute()
	assert.NilError(t, err, "Completion command should run with no errors")
}

/*
func TestCaptureOutput(t *testing.T) {
	cmd := NewCompletionCommand()
	assert.Assert(t, cmd != nil, "Completion command must exist")

	err := cmd.Execute()
	assert.Assert(t, err != nil, "Shell type must be defined")

	output := captureOutput(func() {
		args := []string{"-s", "bash"}
		cmd.SetArgs(args)
		err := cmd.Execute()
		assert.NilError(t, err, "Completion command should run with no errors")
	})

	fmt.Println("=======================================================================================")
	fmt.Println("Output: " + output)
}

func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(os.Stdout)
	return buf.String()
}*/
