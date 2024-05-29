package scarealtime

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/vorpal/vorpalconfiguration"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"gotest.tools/assert"
)

func TestInstallOrUpgrade_firstInstallation_Success(t *testing.T) {
	err := firstInstallation()
	assert.NilError(t, err, "Error on first installation of vorpal")
	fileExists, _ := osinstaller.FileExists(vorpalconfiguration.Params.ExecutableFilePath())
	assert.Assert(t, fileExists, "Executable file not found")
	fileExists, _ = osinstaller.FileExists(vorpalconfiguration.Params.HashFilePath())
	assert.Assert(t, fileExists, "Hash file not found")
}

func firstInstallation() error {
	os.RemoveAll(vorpalconfiguration.Params.WorkingDir())
	err := osinstaller.InstallOrUpgrade(&vorpalconfiguration.Params)
	return err
}

func TestInstallOrUpgrade_installationIsUpToDate_Success(t *testing.T) {
	err := firstInstallation()
	assert.NilError(t, err, "Error on first installation of vorpal")
	err = osinstaller.InstallOrUpgrade(&vorpalconfiguration.Params)
	assert.NilError(t, err, "Error when not need to upgrade")
}

func TestInstallOrUpgrade_installationIsNotUpToDate_Success(t *testing.T) {
	err := firstInstallation()
	assert.NilError(t, err, "Error on first installation of vorpal")
	changeHashFile()
	err = osinstaller.InstallOrUpgrade(&vorpalconfiguration.Params)
	assert.NilError(t, err, "Error when need to upgrade")
	fileExists, _ := osinstaller.FileExists(vorpalconfiguration.Params.ExecutableFilePath())
	assert.Assert(t, fileExists, "Executable file not found")
	fileExists, _ = osinstaller.FileExists(vorpalconfiguration.Params.HashFilePath())
	assert.Assert(t, fileExists, "Hash file not found")
}

func changeHashFile() {
	content, _ := os.ReadFile(vorpalconfiguration.Params.HashFilePath())
	content[0]++
	_ = os.WriteFile(vorpalconfiguration.Params.HashFilePath(), content, os.ModePerm)
}
