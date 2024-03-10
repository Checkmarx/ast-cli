//go:build !integration

package commands

import (
	"testing"

	cliErrors "github.com/checkmarx/ast-cli/internal/errors"
	"gotest.tools/assert"
)

func TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully(t *testing.T) {
	execCmdNilAssertion(t, "import", "--project-name", "my-project", "--import-file-type", "my-type", "--import-file-path", "my-path")
}

func TestImport_ImportSarifFileMissingImportFileType_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--project-name", "my-project", "--import-file-type", "", "--import-file-path", "my-path")
	assert.Assert(t, err.Error() == cliErrors.MissingImportFlags)
}

func TestImport_ImportSarifFileMissingImportFilePath_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--project-name", "my-project", "--import-file-type", "my-type", "--import-file-path", "")
	assert.Assert(t, err.Error() == cliErrors.MissingImportFlags)
}

func TestImport_ImportSarifFileMissingImportProjectName_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--import-file-type", "my-type", "--import-file-path", "my-path")
	assert.Assert(t, err.Error() == cliErrors.ProjectNameIsRequired)
}
