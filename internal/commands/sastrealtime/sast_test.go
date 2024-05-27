package scarealtime

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/sastrealtime/sastconfiguration"
	"github.com/checkmarx/ast-cli/internal/commands/scarealtime/scaconfiguration"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"gotest.tools/assert"
)

func TestSastFirstInstallation(t *testing.T) {
	os.RemoveAll(scaconfiguration.Params.WorkingDir())
	err := osinstaller.InstallOrUpgrade(&sastconfiguration.Params)
	assert.NilError(t, err)
}

func TestSastSecondInstallation(t *testing.T) {
	err := osinstaller.InstallOrUpgrade(&sastconfiguration.Params)
	assert.NilError(t, err)
}
