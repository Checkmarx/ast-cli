//go:build !integration

package util

import (
	"testing"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	cmd := NewImportCommand(
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
		&mock.ByorMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)
	cmd.SetArgs([]string{"utils", "import", "--project-name", "my-project", "--import-file-path", "my-path.sarif"})
	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestImport_ImportSarifFileProjectDoesntExist_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	cmd := NewImportCommand(
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
		&mock.ByorMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)
	cmd.SetArgs([]string{"utils", "import", "--project-name", "MOCK-PROJECT-NOT-EXIST", "--import-file-path", "my-path.sarif"})
	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestImport_ImportSarifFileMissingImportFilePath_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	cmd := NewImportCommand(
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
		&mock.ByorMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)
	cmd.SetArgs([]string{"utils", "import", "--project-name", "my-project"})
	err := cmd.Execute()
	assert.Assert(t, err.Error() == errorConstants.ImportFilePathIsRequired)
}

func TestImport_ImportSarifFileEmptyImportFilePathValue_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	cmd := NewImportCommand(
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
		&mock.ByorMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)
	cmd.SetArgs([]string{"utils", "import", "--project-name", "my-project", "--import-file-path", ""})
	err := cmd.Execute()
	assert.Assert(t, err.Error() == errorConstants.ImportFilePathIsRequired)
}

func TestImport_ImportSarifFileMissingImportProjectName_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	cmd := NewImportCommand(
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
		&mock.ByorMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)
	cmd.SetArgs([]string{"utils", "import", "--import-file-path", "my-path.zip"})
	err := cmd.Execute()
	assert.Assert(t, err.Error() == errorConstants.ProjectNameIsRequired)
}

func TestImport_ImportSarifFileProjectNameNotProvided_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	cmd := NewImportCommand(
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
		&mock.ByorMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)
	cmd.SetArgs([]string{"utils", "import", "--project-name", "", "--import-file-path", "my-path.sarif"})
	err := cmd.Execute()
	assert.Assert(t, err.Error() == errorConstants.ProjectNameIsRequired)
}

func TestImport_ImportSarifFileUnacceptedFileExtension_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	cmd := NewImportCommand(
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
		&mock.ByorMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)
	cmd.SetArgs([]string{"utils", "import", "--project-name", "MOCK-PROJECT-NOT-EXIST", "--import-file-path", "my-path.txt"})
	err := cmd.Execute()
	assert.Assert(t, err.Error() == errorConstants.SarifInvalidFileExtension)
}

func TestImport_ImportSarifFileMissingExtension_CreateImportReturnsErrorWithCorrectMessage(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	cmd := NewImportCommand(
		&mock.ProjectsMockWrapper{},
		&mock.UploadsMockWrapper{},
		&mock.GroupsMockWrapper{},
		mock.AccessManagementMockWrapper{},
		&mock.ByorMockWrapper{},
		mock.ApplicationsMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)
	cmd.SetArgs([]string{"utils", "import", "--project-name", "MOCK-PROJECT-NOT-EXIST", "--import-file-path", "some/path/no/extension/my-path"})
	err := cmd.Execute()
	assert.Assert(t, err.Error() == errorConstants.SarifInvalidFileExtension)
}

func TestImporFileFunction_FakeUnauthorizedHttpStatusCode_ReturnRelevantError(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	err := importFile(mock.FakeUnauthorized401, "importFilePath", &mock.UploadsMockWrapper{}, &mock.ByorMockWrapper{}, &mock.FeatureFlagsMockWrapper{})
	assert.Assert(t, err.Error() == errorConstants.StatusUnauthorized)
}

func TestImporFileFunction_FakeForbiddenHttpStatusCode_ReturnRelevantError(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	err := importFile(mock.FakeForbidden403, "importFilePath", &mock.UploadsMockWrapper{}, &mock.ByorMockWrapper{}, &mock.FeatureFlagsMockWrapper{})
	assert.Assert(t, err.Error() == errorConstants.StatusForbidden)
}

func TestImporFileFunction_FakeInternalServerErrorHttpStatusCode_ReturnRelevantError(t *testing.T) {
	wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] = true
	err := importFile(mock.FakeInternalServerError500, "importFilePath", &mock.UploadsMockWrapper{}, &mock.ByorMockWrapper{}, &mock.FeatureFlagsMockWrapper{})
	assert.Assert(t, err.Error() == errorConstants.StatusInternalServerError)
}
