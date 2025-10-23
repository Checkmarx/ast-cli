//go:build integration

package integration

import (
	"strings"
	"testing"

	"gotest.tools/assert"
)

// Help
/*
This function helps to validate all the cx --help command content.
Expected help content value is stored in the "integration/data/cxHelpText.txt"
We compare the command output with the above txt file, if there is any new flag introduced
or content is changed then user this testcase will help to capture it
*/
func TestHelpFlag_Validate_CxHelpOutput(t *testing.T) {
	referenceFile := "data/console-help-text-log/cxHelpText.txt"

	_, outputText := executeCommand(t, "help")

	ValidateCompleteConsoleLog(t, outputText, referenceFile)
}

// Auth
// Validate cx auth register --help command
func TestHelpFlag_Validate_AuthRegisterHelpMessage(t *testing.T) {

	args := []string{
		"auth",
		"register",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Register new OAuth2 client and outputs its generated credentials in the format <key>=<value>", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx auth validate --help command
func TestHelpFlag_Validate_AuthValidateHelpMessage(t *testing.T) {

	args := []string{
		"auth",
		"validate",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Validates if CLI is able to communicate with Checkmarx One", textCapturedForValidation, "Incorrect help text found")
}

// Configure

// Validate cx configure --help command
func TestHelpFlag_Validate_ConfigureHelpMessage(t *testing.T) {

	args := []string{
		"configure",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The configure command is the fastest way to set up your AST CLI", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx configure set --help command
func TestHelpFlag_Validate_ConfigureSetHelpMessage(t *testing.T) {

	args := []string{
		"configure",
		"set",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Set configuration properties", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx configure show --help command
func TestHelpFlag_Validate_ConfigureShowHelpMessage(t *testing.T) {

	args := []string{
		"configure",
		"show",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Shows effective profile configuration", textCapturedForValidation, "Incorrect help text found")
}

// Hooks

// Validate cx configure show --help command
func TestHelpFlag_Validate_HooksPreCommitHelpMessage(t *testing.T) {

	args := []string{
		"hooks",
		"pre-commit",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The pre-commit command enables the ability to manage Git pre-commit hooks for secret detection.", textCapturedForValidation, "Incorrect help text found")
}

// Project Help Validation

// Validate cx project list --help command
func TestHelpFlag_ValidateProjectListHelpMessage(t *testing.T) {

	args := []string{
		"project",
		"list",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "List all projects in the system", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx project create --help
func TestHelpFlag_ValidateProjectCreateHelpMessage(t *testing.T) {

	referenceFile := "data/console-help-text-log/projectCreateHelpText.txt"

	args := []string{
		"project",
		"create",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	ValidateCompleteConsoleLog(t, outputText, referenceFile)
}

// Validate cx project delete --help command
func TestHelpFlag_ValidateProjectDeleteHelpMessage(t *testing.T) {

	args := []string{
		"project",
		"delete",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Delete a project", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx project show --help command
func TestHelpFlag_ValidateProjectShowHelpMessage(t *testing.T) {

	args := []string{
		"project",
		"show",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Show information about a project", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx project tags --help command
func TestHelpFlag_ValidateProjectTagsHelpMessage(t *testing.T) {

	args := []string{
		"project",
		"tags",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Get a list of all available tags", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx project branch --help command
func TestHelpFlag_ValidateProjectBranchHelpMessage(t *testing.T) {

	args := []string{
		"project",
		"branches",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Show list of branches from a project", textCapturedForValidation, "Incorrect help text found")
}

// Results

// Validate cx results --help command
func TestHelpFlag_Validate_ResultsHelpMessage(t *testing.T) {

	args := []string{
		"results",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Retrieve results", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx results show --help command
func TestHelpFlag_Validate_ResultsShowHelpOutput(t *testing.T) {
	referenceFile := "data/console-help-text-log/resultsShowHelpLog.txt"

	args := []string{
		"results",
		"show",
		"--help",
	}
	_, outputText := executeCommand(t, args...)

	ValidateCompleteConsoleLog(t, outputText, referenceFile)
}

// Validate cx results codebashing --help command
func TestHelpFlag_Validate_ResultsCodeBashingHelpMessage(t *testing.T) {

	args := []string{
		"results",
		"codebashing",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The codebashing command enables the ability to retrieve the link about a specific vulnerability", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx results exit-code --help command
func TestHelpFlag_Validate_ResultsExitCodeHelpMessage(t *testing.T) {

	args := []string{
		"results",
		"exit-code",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The exit-code command enables you to get the exit code and failure details of a requested scan in Checkmarx One", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx triage --help command
func TestHelpFlag_Validate_TriageHelpMessage(t *testing.T) {

	args := []string{
		"triage",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The 'triage' command enables the ability to manage results in Checkmarx One", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx triage get-states --help command
func TestHelpFlag_Validate_TriageGetStatesHelpMessage(t *testing.T) {

	args := []string{
		"triage",
		"get-states",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The get-states command shows information about each of the custom states that have been configured in your tenant account", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx triage update --help command
func TestHelpFlag_Validate_TriageUpdateHelpMessage(t *testing.T) {

	args := []string{
		"triage",
		"update",
		"--help",
	}
	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The update command enables the ability to triage the results in Checkmarx One", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx triage show --help command
func TestHelpFlag_Validate_TriageShowHelpMessage(t *testing.T) {

	args := []string{
		"triage",
		"show",
		"--help",
	}
	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The show command provides a list of all the predicates in the issue", textCapturedForValidation, "Incorrect help text found")
}

// Scan Help Validation

// Validate cx scan cancel --help command
func TestHelpFlag_Validate_ScanCancelHelpMessage(t *testing.T) {

	args := []string{
		"scan",
		"cancel",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The cancel command enables the ability to cancel one or more running scans in Checkmarx One", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx scan create --help command
func TestHelpFlag_Validate_CxScanCreateHelpOutput(t *testing.T) {
	referenceFile := "data/console-help-text-log/scanCreateHelpLog.txt"

	args := []string{
		"scan",
		"create",
		"--help",
	}
	_, outputText := executeCommand(t, args...)

	ValidateCompleteConsoleLog(t, outputText, referenceFile)
}

// Validate cx scan delete --help command
func TestHelpFlag_Validate_ScanDeleteHelpMessage(t *testing.T) {

	args := []string{
		"scan",
		"delete",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Deletes one or more scans", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx scan list --help command
func TestHelpFlag_Validate_ScanListHelpMessage(t *testing.T) {

	args := []string{
		"scan",
		"list",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The list command provides a list of all the scans in Checkmarx One", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx tags show --help command
func TestHelpFlag_Validate_ScanShowHelpMessage(t *testing.T) {

	args := []string{
		"scan",
		"show",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The show command enables the ability to show information about a requested scan in Checkmarx One", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx scan tags --help command
func TestHelpFlag_Validate_ScanTagsHelpMessage(t *testing.T) {

	args := []string{
		"scan",
		"tags",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The tags command enables the ability to provide a list of all the available tags in Checkmarx One", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx scan workflow --help command
func TestHelpFlag_Validate_ScanWorkflowHelpMessage(t *testing.T) {

	args := []string{
		"scan",
		"workflow",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The workflow command enables the ability to provide information about a requested scan workflow in Checkmarx One", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx scan logs --help command
func TestHelpFlag_Validate_ScanLogsHelpMessage(t *testing.T) {

	args := []string{
		"scan",
		"logs",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "Accepts a scan-id and scan type (sast, iac-security) and downloads the related scan log", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx scan sca-realtime --help command
func TestHelpFlag_Validate_ScanScaRealtimeHelpMessage(t *testing.T) {

	args := []string{
		"scan",
		"sca-realtime",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The sca-realtime command enables the ability to create, run and retrieve results from a sca scan using sca resolver", textCapturedForValidation, "Incorrect help text found")
}

// Validate cx scan kics-realtime --help command
func TestHelpFlag_Validate_ScanKicsRealtimeHelpMessage(t *testing.T) {

	args := []string{
		"scan",
		"kics-realtime",
		"--help",
	}

	_, outputText := executeCommand(t, args...)

	normalizedOut := StripAnsi(strings.ReplaceAll(outputText.String(), "\r\n", "\n"))
	textCapturedForValidation := GetFlagHelpText(normalizedOut)

	assert.Equal(t, "The kics-realtime command enables the ability to create, run and retrieve results from a kics scan using a docker image", textCapturedForValidation, "Incorrect help text found")
}
