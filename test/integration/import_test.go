//go:build integration

package integration

import (
	"testing"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully(t *testing.T) {
	projectId, projectName := createProject(t, nil, nil)
	defer deleteProject(t, projectId)
	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\sarif.sarif",
	}
	err, _ := executeCommand(t, args...)
	defer deleteProjectByName(t, projectName)
	assert.NilError(t, err, "Import command failed with existing project")
}

func TestImport_ImportSarifFileWithCorrectFlagsZipFileExtention_CreateImportSuccessfully(t *testing.T) {
	projectId, projectName := createProject(t, nil, nil)
	defer deleteProject(t, projectId)
	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\sarif.zip",
	}
	err, _ := executeCommand(t, args...)
	defer deleteProjectByName(t, projectName)
	assert.NilError(t, err, "Import command failed with existing project")
}

func TestImport_ImportSarifFileProjectDoesntExist_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {
	projectName := projectNameRandom

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\sarif.sarif",
	}
	err, _ := executeCommand(t, args...)

	defer deleteProjectByName(t, projectName)

	assert.NilError(t, err, "Import command failed with non-existing project")
}

func TestImport_ImportSarifFileMissingVersion_ImportFailWithCorrectMessage(t *testing.T) {
	projectName := projectNameRandom

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\sarif-missing-version.sarif",
	}
	err, _ := executeCommand(t, args...)
	defer deleteProjectByName(t, projectName)
	assertError(t, err, errorConstants.ImportSarifFileErrorMessage)
}

func TestImport_ImportMalformedSarifFile_ImportFailWithCorrectMessage(t *testing.T) {
	projectName := projectNameRandom

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), ".\\data\\malformed-sarif.sarif",
	}
	err, _ := executeCommand(t, args...)
	deleteProjectByName(t, projectName)
	assertError(t, err, errorConstants.ImportSarifFileErrorMessage)
}
