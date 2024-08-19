//go:build integration

package integration

import (
	"fmt"
	"testing"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully(t *testing.T) {

	projectId, projectName := CreateProject(t, nil, nil)
	defer DeleteProject(t, projectId)

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/sarif.sarif",
	}
	err, _ := executeCommand(t, args...)

	assert.NilError(t, err, "Import command failed with existing project")
}

func TestImport_ImportSarifFileWithCorrectFlagsZipFileExtention_CreateImportSuccessfully(t *testing.T) {

	projectId, projectName := CreateProject(t, nil, nil)
	defer DeleteProject(t, projectId)

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/sarif.zip",
	}
	err, _ := executeCommand(t, args...)

	assert.NilError(t, err, "Import command failed with existing project")
}

func TestImport_ImportSarifFileProjectDoesntExist_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {

	projectName := projectNameRandom
	defer DeleteProjectByName(t, projectName)

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/sarif.sarif",
	}
	err, _ := executeCommand(t, args...)

	assert.NilError(t, err, "Import command failed with non-existing project")
}

func TestImport_ImportSarifFileMissingVersion_ImportFailWithCorrectMessage(t *testing.T) {

	projectName := projectNameRandom

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/sarif-missing-version.sarif",
	}
	err, _ := executeCommand(t, args...)

	DeleteProjectByName(t, projectName)

	assertError(t, err, fmt.Sprintf(errorConstants.ImportSarifFileErrorMessageWithMessage, 400, ""))
}

func TestImport_ImportMalformedSarifFile_ImportFailWithCorrectMessage(t *testing.T) {

	projectName := projectNameRandom

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "data/malformed-sarif.sarif",
	}
	err, _ := executeCommand(t, args...)
	defer DeleteProjectByName(t, projectName)
	assertError(t, err, fmt.Sprintf(errorConstants.ImportSarifFileErrorMessageWithMessage, 400, ""))
}

func TestImport_MissingImportFlag_ImportFailWithCorrectMessage(t *testing.T) {

	projectName := projectNameRandom

	args := []string{
		"utils", "import",
		flag(params.ProjectName), projectName,
	}
	err, _ := executeCommand(t, args...)
	DeleteProjectByName(t, projectName)
	assertError(t, err, errorConstants.ImportFilePathIsRequired)
}

func TestGetProjectNameFunction_ProjectNameValueIsEmpty_ReturnRelevantError(t *testing.T) {

	args := []string{
		"utils", "import",
		flag(params.ProjectName), "",
		flag(params.ImportFilePath), "data/sarif.sarif",
	}
	err, _ := executeCommand(t, args...)
	assertError(t, err, errorConstants.ProjectNameIsRequired)
}
