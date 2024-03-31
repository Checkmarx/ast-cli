//go:build integration

package integration

import (
	"testing"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

func TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully(t *testing.T) {
	// createASTIntegrationTestCommand is called just to load the FF values
	createASTIntegrationTestCommand(t)
	if !wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] {
		t.Skip("BYOR flag is currently disabled")
	}

	projectId, projectName := createProject(t, nil, nil)
	defer deleteProject(t, projectId)

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\sarif.sarif",
	}
	err, _ := executeCommand(t, args...)

	assert.NilError(t, err, "Import command failed with existing project")
}

func TestImport_ImportSarifFileWithCorrectFlagsZipFileExtention_CreateImportSuccessfully(t *testing.T) {
	// createASTIntegrationTestCommand is called just to load the FF values
	createASTIntegrationTestCommand(t)
	if !wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] {
		t.Skip("BYOR flag is currently disabled")
	}

	projectId, projectName := createProject(t, nil, nil)
	defer deleteProject(t, projectId)

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\sarif.zip",
	}
	err, _ := executeCommand(t, args...)

	assert.NilError(t, err, "Import command failed with existing project")
}

func TestImport_ImportSarifFileProjectDoesntExist_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {
	// createASTIntegrationTestCommand is called just to load the FF values
	createASTIntegrationTestCommand(t)
	if !wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] {
		t.Skip("BYOR flag is currently disabled")
	}

	projectName := projectNameRandom
	defer deleteProjectByName(t, projectName)

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\sarif.sarif",
	}
	err, _ := executeCommand(t, args...)

	assert.NilError(t, err, "Import command failed with non-existing project")
}

func TestImport_ImportSarifFileMissingVersion_ImportFailWithCorrectMessage(t *testing.T) {
	// createASTIntegrationTestCommand is called just to load the FF values
	createASTIntegrationTestCommand(t)
	if !wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] {
		t.Skip("BYOR flag is currently disabled")
	}

	projectName := projectNameRandom
	defer deleteProjectByName(t, projectName)

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\sarif-missing-version.sarif",
	}
	err, _ := executeCommand(t, args...)

	assertError(t, err, errorConstants.ImportSarifFileErrorMessage)
}

func TestImport_ImportMalformedSarifFile_ImportFailWithCorrectMessage(t *testing.T) {
	// createASTIntegrationTestCommand is called just to load the FF values
	createASTIntegrationTestCommand(t)
	if !wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] {
		t.Skip("BYOR flag is currently disabled")
	}

	projectName := projectNameRandom

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\malformed-sarif.sarif",
	}
	err, _ := executeCommand(t, args...)
	defer deleteProjectByName(t, projectName)
	assertError(t, err, errorConstants.ImportSarifFileErrorMessage)
}

func TestImport_MissingImportFlag_ImportFailWithCorrectMessage(t *testing.T) {
	// createASTIntegrationTestCommand is called just to load the FF values
	createASTIntegrationTestCommand(t)
	if !wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] {
		t.Skip("BYOR flag is currently disabled")
	}

	featureFlagsPath := viper.GetString(params.FeatureFlagsKey)
	_ = wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)

	projectName := projectNameRandom

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
	}
	err, _ := executeCommand(t, args...)
	deleteProjectByName(t, projectName)
	assertError(t, err, errorConstants.ImportFilePathIsRequired)
}
