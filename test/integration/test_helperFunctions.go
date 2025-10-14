package integration

import (
	"bytes"
	"gotest.tools/assert"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"testing"
)

/*
When cli logs the output in console it prints some texts in the special format
e.g. [1mCOMMANDS[0m this methos helps to strip this special format from the output
*/
func Strip_ANSI(s string) string {
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	return ansi.ReplaceAllString(s, "")
}

// Returns the first line of information when --help flag is passed along a command
func GetFlagHelpText(s string) string {

	linesSepration := strings.SplitN(s, "\n", 2)
	textCapturedForValidation := strings.TrimSpace(linesSepration[0])

	return textCapturedForValidation
}

// Compares the complete console log output against the given text file data
func ValidateCompleteConsoleLog(t *testing.T, consoleLog *bytes.Buffer, filePath string) {
	//Read the reference file data
	referenceData, err := ioutil.ReadFile(filePath)

	if err != nil {
		log.Fatalf("Error reading help text: %s", err)
	}
	//formats console output and reference file data
	normalizedRef := Strip_ANSI(strings.ReplaceAll(string(referenceData), "\r\n", "\n"))
	normalizedOut := Strip_ANSI(strings.ReplaceAll(consoleLog.String(), "\r\n", "\n"))

	assert.Equal(t, normalizedRef, normalizedOut, "Command output doesn't match with given file")
}
