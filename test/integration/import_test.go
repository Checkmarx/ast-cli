//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully(t *testing.T) {
	projectId, projectName := createProject(t, nil, nil)
	defer deleteProject(t, projectId)
	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "./data/sarif.sarif",
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "import failed")
}

func TestImport_ImportSarifFileProjectDoesntExist_CreateImportWithProvidedNewNameSuccessfully(t *testing.T) {
	projectName := projectNameRandom()

	args := []string{
		"import",
		flag(params.ProjectName), projectName,
		flag(params.ImportFilePath), "./data/sarif.sarif",
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "import failed")
}
