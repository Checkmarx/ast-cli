//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestImport_providedProjectDoesNotExist_correctError(t *testing.T) {
	args := []string{
		"import",
		flag(params.ProjectName), "ProjectDoesNotExist",
		flag(params.ImportFilePath), "./data/sarif.sarif",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Project name does not exist")
}
func TestImport_okFromByorImports_projectContainsImportedData(t *testing.T) {
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
