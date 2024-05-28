package scarealtime

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/vorpal/vorpalconfiguration"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"gotest.tools/assert"
)

func TestInstallOrUpgrade_firstInstallation_Success(t *testing.T) {
	os.RemoveAll(vorpalconfiguration.Params.WorkingDir())
	err := osinstaller.InstallOrUpgrade(&vorpalconfiguration.Params)
	assert.NilError(t, err)
	dirEntry, err := os.ReadDir(vorpalconfiguration.Params.WorkingDir())
	assert.NilError(t, err, "Error reading directory ", vorpalconfiguration.Params.WorkingDir())
	assert.Assert(t, dirContainsFile(dirEntry, vorpalconfiguration.Params.ExecutableFile))
	assert.Assert(t, dirContainsFile(dirEntry, vorpalconfiguration.Params.HashFileName))
}

func dirContainsFile(entry []os.DirEntry, file string) bool {
	for _, e := range entry {
		if e.Name() == file {
			return true
		}
	}
	return false
}

func TestInstallOrUpgrade_installationIsUpToDate_Success(t *testing.T) {
	err := osinstaller.InstallOrUpgrade(&vorpalconfiguration.Params)
	assert.NilError(t, err)
}

func TestInstallOrUpgrade_installationIsNotUpToDate_Success(t *testing.T) {
	changeHashFile()
	err := osinstaller.InstallOrUpgrade(&vorpalconfiguration.Params)
	assert.NilError(t, err)
	dirEntry, err := os.ReadDir(vorpalconfiguration.Params.WorkingDir())
	assert.NilError(t, err, "Error reading directory ", vorpalconfiguration.Params.WorkingDir())
	assert.Assert(t, dirContainsFile(dirEntry, vorpalconfiguration.Params.ExecutableFile))
	assert.Assert(t, dirContainsFile(dirEntry, vorpalconfiguration.Params.HashFileName))
}

func changeHashFile() {
	content, _ := os.ReadFile(vorpalconfiguration.Params.HashFilePath())
	content[0]++
	os.WriteFile(vorpalconfiguration.Params.HashFilePath(), content, os.ModePerm)
}
