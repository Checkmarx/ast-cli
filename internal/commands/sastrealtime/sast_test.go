package scarealtime

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/sastrealtime/sastconfiguration"
	"github.com/checkmarx/ast-cli/internal/commands/scarealtime/scaconfiguration"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"gotest.tools/assert"
)

func TestInstallOrUpgrade_firstInstallation_Success(t *testing.T) {
	os.RemoveAll(scaconfiguration.Params.WorkingDir())
	err := osinstaller.InstallOrUpgrade(&sastconfiguration.Params)
	assert.NilError(t, err)
	dirEntry, err := os.ReadDir(scaconfiguration.Params.WorkingDir())
	assert.NilError(t, err, "Error reading directory ", scaconfiguration.Params.WorkingDir())
	assert.Assert(t, dirEntry != nil)
}

func TestInstallOrUpgrade_secondInstallation_Success(t *testing.T) {
	err := osinstaller.InstallOrUpgrade(&sastconfiguration.Params)
	assert.NilError(t, err)
}
