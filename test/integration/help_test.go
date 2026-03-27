//go:build integration

package integration

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"gotest.tools/assert"
)

// verify cx --help
// verify cx project --help
// verify cx scan --help
// verify cx triage --help
// verify cx results --help
func TestCxRootHelpCommand(t *testing.T) {
	// Capture the output of the help command
	buffer := executeCmdNilAssertion(
		t,
		"--help",
	)

	// Read the output
	result, err := io.ReadAll(buffer)
	assert.NilError(t, err, "Reading help command output should succeed")

	// Convert output to string and check for expected content
	output := string(result)
	fmt.Println("Help output:\n", output)

	assert.Assert(t, strings.Contains(output, "The Checkmarx One CLI is a fully functional Command Line Interface (CLI) that interacts with the Checkmarx One server"))
	assert.Assert(t, strings.Contains(output, "cx <command> <subcommand> [flags]"))
}

func TestProjectHelpCommand(t *testing.T) {
	// Capture the output of the help command
	buffer := executeCmdNilAssertion(
		t,
		"The help command for 'project' should execute successfully",
		"project", "--help",
	)

	// Read the output
	result, err := io.ReadAll(buffer)
	assert.NilError(t, err, "Reading help command output should succeed")

	// Convert output to string and check for expected content
	output := string(result)
	fmt.Println("Help project output:\n", output)

	// Assert it contains expected help output
	assert.Assert(t, strings.Contains(output, "The project command enables the ability to manage projects in Checkmarx One"), "Help output should contain command description")
	assert.Assert(t, strings.Contains(output, "cx project [flags]"), "Help output should contain usage information")
	assert.Assert(t, strings.Contains(output, "COMMANDS"), "Help output should list available COMMANDS")
}

func TestScanHelpCommand(t *testing.T) {
	// Capture the output of the help command
	buffer := executeCmdNilAssertion(
		t,
		"The help command for 'scan' should execute successfully",
		"scan", "--help",
	)

	// Read the output
	result, err := io.ReadAll(buffer)
	assert.NilError(t, err, "Reading help command output should succeed")

	// Convert output to string and check for expected content
	output := string(result)
	fmt.Println("Help scan output:\n", output)

	// Assert it contains some expected help output
	assert.Assert(t, strings.Contains(output, "The scan command enables the ability to manage scans in Checkmarx One"), "Help output should contain command description")
	assert.Assert(t, strings.Contains(output, "cx scan [flags]"), "Help output should contain usage information")
	assert.Assert(t, strings.Contains(output, "COMMANDS"), "Help output should list available COMMANDS")
}

func TestTriageHelpCommand(t *testing.T) {
	// Capture the output of the help command
	buffer := executeCmdNilAssertion(
		t,
		"The help command for 'triage' should execute successfully",
		"triage", "--help",
	)

	// Read the output
	result, err := io.ReadAll(buffer)
	assert.NilError(t, err, "Reading help command output should succeed")

	// Convert output to string and check for expected content
	output := string(result)
	fmt.Println("Help triage output:\n", output)

	// Assert it contains expected help output
	assert.Assert(t, strings.Contains(output, "The 'triage' command enables the ability to manage results in Checkmarx One"), "Help output should contain command description")
	assert.Assert(t, strings.Contains(output, "cx triage [flags]"), "Help output should contain usage information")
	assert.Assert(t, strings.Contains(output, "COMMANDS"), "Help output should list available COMMANDS")
}

func TestResultsHelpCommand(t *testing.T) {
	// Capture the output of the help command
	buffer := executeCmdNilAssertion(
		t,
		"The help command for 'results' should execute successfully",
		"results", "--help",
	)

	// Read the output
	result, err := io.ReadAll(buffer)
	assert.NilError(t, err, "Reading help command output should succeed")

	// Convert output to string and check for expected content
	output := string(result)
	fmt.Println("Help results output:\n", output)

	// Assert it contains expected help output
	assert.Assert(t, strings.Contains(output, "Retrieve results"), "Help output should contain command description")
	assert.Assert(t, strings.Contains(output, "cx results [flags]"), "Help output should contain usage information")
	assert.Assert(t, strings.Contains(output, "COMMANDS"), "Help output should list available COMMANDS")
}
