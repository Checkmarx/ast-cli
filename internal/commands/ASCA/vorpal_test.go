package asca

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/ASCA/ASCAconfig"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"gotest.tools/assert"
)

func TestInstallOrUpgrade_firstInstallation_Success(t *testing.T) {
	err := firstInstallation()
	assert.NilError(t, err, "Error on first installation of ASCA")
	fileExists, _ := osinstaller.FileExists(ascaconfig.Params.ExecutableFilePath())
	assert.Assert(t, fileExists, "Executable file not found")
	fileExists, _ = osinstaller.FileExists(ascaconfig.Params.HashFilePath())
	assert.Assert(t, fileExists, "Hash file not found")
}

func firstInstallation() error {
	os.RemoveAll(ascaconfig.Params.WorkingDir())
	_, err := osinstaller.InstallOrUpgrade(&ascaconfig.Params)
	return err
}

func TestInstallOrUpgrade_installationIsUpToDate_Success(t *testing.T) {
	err := firstInstallation()
	assert.NilError(t, err, "Error on first installation of ASCA")
	_, err = osinstaller.InstallOrUpgrade(&ascaconfig.Params)
	assert.NilError(t, err, "Error when not need to upgrade")
}

func TestInstallOrUpgrade_installationIsNotUpToDate_Success(t *testing.T) {
	err := firstInstallation()
	assert.NilError(t, err, "Error on first installation of ASCA")
	changeHashFile()
	_, err = osinstaller.InstallOrUpgrade(&ascaconfig.Params)
	assert.NilError(t, err, "Error when need to upgrade")
	fileExists, _ := osinstaller.FileExists(ascaconfig.Params.ExecutableFilePath())
	assert.Assert(t, fileExists, "Executable file not found")
	fileExists, _ = osinstaller.FileExists(ascaconfig.Params.HashFilePath())
	assert.Assert(t, fileExists, "Hash file not found")
}

func changeHashFile() {
	content, _ := os.ReadFile(ascaconfig.Params.HashFilePath())
	content[0]++
	_ = os.WriteFile(ascaconfig.Params.HashFilePath(), content, os.ModePerm)
}
