//go:build integration

package integration

import (
	"fmt"
	"testing"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"github.com/checkmarx/ast-cli/internal/params"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully(t *testing.T) {
	if !isFFByorEnabled(t) {
		assert.Assert(t, true, "Byor is disabled")
		return
	}
	projectId, projectName := createProject(t, nil, nil)
	defer deleteProject(t, projectId)

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/sarif.sarif",
	}
	err, _ := executeCommand(t, args...)

	assert.NilError(t, err, "Import command failed with existing project")
}

func TestImport_ImportSarifFileWithCorrectFlagsZipFileExtention_CreateImportSuccessfully(t *testing.T) {
	if !isFFByorEnabled(t) {
		assert.Assert(t, true, "Byor is disabled")
		return
	}

	projectId, projectName := createProject(t, nil, nil)
	defer deleteProject(t, projectId)

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/sarif.zip",
	}
	err, _ := executeCommand(t, args...)

	assert.NilError(t, err, "Import command failed with existing project")
}

func TestImport_ImportSarifFileProjectDoesntExist_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {
	if !isFFByorEnabled(t) {
		assert.Assert(t, true, "Byor is disabled")
		return
	}

	projectName := projectNameRandom
	defer deleteProjectByName(t, projectName)

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/sarif.sarif",
	}
	err, _ := executeCommand(t, args...)

	assert.NilError(t, err, "Import command failed with non-existing project")
}

func TestImport_ImportSarifFileMissingVersion_ImportFailWithCorrectMessage(t *testing.T) {
	if !isFFByorEnabled(t) {
		assert.Assert(t, true, "Byor is disabled")
		return
	}

	projectName := projectNameRandom

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/sarif-missing-version.sarif",
	}
	err, _ := executeCommand(t, args...)

	deleteProjectByName(t, projectName)

	assertError(t, err, fmt.Sprintf(errorConstants.ImportSarifFileErrorMessageWithMessage, 400, ""))
}

func TestImport_ImportMalformedSarifFile_ImportFailWithCorrectMessage(t *testing.T) {
	if !isFFByorEnabled(t) {
		assert.Assert(t, true, "Byor is disabled")
		return
	}

	projectName := projectNameRandom

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/malformed-sarif.sarif",
	}
	err, _ := executeCommand(t, args...)
	defer deleteProjectByName(t, projectName)
	assertError(t, err, fmt.Sprintf(errorConstants.ImportSarifFileErrorMessageWithMessage, 400, ""))
}

func TestImport_MissingImportFlag_ImportFailWithCorrectMessage(t *testing.T) {
	if !isFFByorEnabled(t) {
		assert.Assert(t, true, "Byor is disabled")
		return
	}

	projectName := projectNameRandom

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
	}
	err, _ := executeCommand(t, args...)
	deleteProjectByName(t, projectName)
	assertError(t, err, errorConstants.ImportFilePathIsRequired)
}

func TestGetProjectNameFunction_ProjectNameValueIsEmpty_ReturnRelevantError(t *testing.T) {
	if !isFFByorEnabled(t) {
		assert.Assert(t, true, "Byor is disabled")
		return
	}

	args := []string{
		"utils", "import",
		flag(params.ProjectName), "",
		flag(params.ImportFilePath), "data/sarif.sarif",
	}
	err, _ := executeCommand(t, args...)
	assertError(t, err, errorConstants.ProjectNameIsRequired)
}

func isFFByorEnabled(t *testing.T) bool {
	// createASTIntegrationTestCommand is called just to load the FF values
	createASTIntegrationTestCommand(t)

	featureFlagsPath := viper.GetString(commonParams.FeatureFlagsKey)
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)

	flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, featureFlagsConstants.ByorEnabled)
	return flagResponse.Status
}
