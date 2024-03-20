//go:build !integration

package commands

import (
	"testing"

	errors2 "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully(t *testing.T) {
	execCmdNilAssertion(t, "import", "--project-name", "my-project", "--import-file-path", "my-path.sarif")
}

func TestImport_ImportSarifFileProjectDoesntExist_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {
	execCmdNilAssertion(t, "import", "--project-name", "MOCK-PROJECT-NOT-EXIST", "--import-file-path", "my-path.sarif")
}

func TestImport_ImportSarifFileMissingImportFilePath_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--project-name", "my-project", "--import-file-path", "")
	assert.Assert(t, err.Error() == errors2.ImportFilePathIsRequired)
}

func TestImport_ImportSarifFileMissingImportProjectName_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--import-file-path", "my-path.zip")
	assert.Assert(t, err.Error() == errors2.ProjectNameIsRequired)
}

func TestImport_ImportSarifFileProjectNameNotProvided_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--project-name", "", "--import-file-path", "my-path.sarif")
	assert.Assert(t, err.Error() == errors2.ProjectNameIsRequired)
}

func TestImport_ImportSarifFileUnacceptedFileExtension_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--project-name", "MOCK-PROJECT-NOT-EXIST", "--import-file-path", "my-path.txt")
	assert.Assert(t, err.Error() == errors2.SarifInvalidFileExtension)
}

func TestImport_ImportSarifFileMissingExtension_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	err := execCmdNotNilAssertion(t, "import", "--project-name", "MOCK-PROJECT-NOT-EXIST", "--import-file-path", "some/path/no/extension/my-path")
	assert.Assert(t, err.Error() == errors2.SarifInvalidFileExtension)
}

func TestImporFileFunction_FakeUnauthorizedHttpStatusCode_ReturnRelevantError(t *testing.T) {
	err := importFile(mock.FakeUnauthorized401, "importFilePath", &mock.UploadsMockWrapper{}, &mock.ByorMockWrapper{})
	assert.Assert(t, err.Error() == errors2.StatusUnauthorized)
}

func TestImporFileFunction_FakeForbiddenHttpStatusCode_ReturnRelevantError(t *testing.T) {
	err := importFile(mock.FakeForbidden403, "importFilePath", &mock.UploadsMockWrapper{}, &mock.ByorMockWrapper{})
	assert.Assert(t, err.Error() == errors2.StatusForbidden)
}

func TestImporFileFunction_FakeInternalServerErrorHttpStatusCode_ReturnRelevantError(t *testing.T) {
	err := importFile(mock.FakeInternalServerError500, "importFilePath", &mock.UploadsMockWrapper{}, &mock.ByorMockWrapper{})
	assert.Assert(t, err.Error() == errors2.StatusInternalServerError)
}