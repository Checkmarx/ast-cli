package vorpal

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/vorpal/vorpalconfig"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"gotest.tools/assert"
)

func TestInstallOrUpgrade_firstInstallation_Success(t *testing.T) {
	err := firstInstallation()
	assert.NilError(t, err, "Error on first installation of vorpal")
	fileExists, _ := osinstaller.FileExists(vorpalconfig.Params.ExecutableFilePath())
	assert.Assert(t, fileExists, "Executable file not found")
	fileExists, _ = osinstaller.FileExists(vorpalconfig.Params.HashFilePath())
	assert.Assert(t, fileExists, "Hash file not found")
}

func firstInstallation() error {
	os.RemoveAll(vorpalconfig.Params.WorkingDir())
	_, err := osinstaller.InstallOrUpgrade(&vorpalconfig.Params)
	return err
}

func TestInstallOrUpgrade_installationIsUpToDate_Success(t *testing.T) {
	err := firstInstallation()
	assert.NilError(t, err, "Error on first installation of vorpal")
	_, err = osinstaller.InstallOrUpgrade(&vorpalconfig.Params)
	assert.NilError(t, err, "Error when not need to upgrade")
}

func TestInstallOrUpgrade_installationIsNotUpToDate_Success(t *testing.T) {
	err := firstInstallation()
	assert.NilError(t, err, "Error on first installation of vorpal")
	changeHashFile()
	_, err = osinstaller.InstallOrUpgrade(&vorpalconfig.Params)
	assert.NilError(t, err, "Error when need to upgrade")
	fileExists, _ := osinstaller.FileExists(vorpalconfig.Params.ExecutableFilePath())
	assert.Assert(t, fileExists, "Executable file not found")
	fileExists, _ = osinstaller.FileExists(vorpalconfig.Params.HashFilePath())
	assert.Assert(t, fileExists, "Hash file not found")
}

func changeHashFile() {
	content, _ := os.ReadFile(vorpalconfig.Params.HashFilePath())
	content[0]++
	_ = os.WriteFile(vorpalconfig.Params.HashFilePath(), content, os.ModePerm)
}
