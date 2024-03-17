//go:build !integration

package commands

import (
	"testing"

	errorconsts "github.com/checkmarx/ast-cli/internal/constants"
	"gotest.tools/assert"
)

func TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully(t *testing.T) {
	execCmdNilAssertion(t, "import", "--project-name", "my-project", "--import-file-path", "my-path.sarif")
}

func TestImport_ImportSarifFileMissingImportFilePath_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--project-name", "my-project", "--import-file-path", "")
	assert.Assert(t, err.Error() == errorconsts.ImportFilePathIsRequired)
}

func TestImport_ImportSarifFileMissingImportProjectName_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--import-file-path", "my-path.zip")
	assert.Assert(t, err.Error() == errorconsts.ProjectNameIsRequired)
}

func TestImport_ImportSarifFileProjectDoesntExist_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {
	execCmdNilAssertion(t, "import", "--project-name", "MOCK-PROJECT-NOT-EXIST", "--import-file-path", "my-path.sarif")
}

func TestImport_ImportSarifFileUnacceptedFileExtension_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--project-name", "MOCK-PROJECT-NOT-EXIST", "--import-file-path", "my-path.txt")
	assert.Assert(t, err.Error() == errorconsts.SarifInvalidFileExtension)
}

func TestImport_ImportSarifFileMissingExtension_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--project-name", "MOCK-PROJECT-NOT-EXIST", "--import-file-path", "some/path/no/extension/my-path")
	assert.Assert(t, err.Error() == errorconsts.SarifInvalidFileExtension)
}
