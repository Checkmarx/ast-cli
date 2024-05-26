package scarealtime

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/sastrealtime/sastconfiguration"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"gotest.tools/assert"
)

func TestSast(t *testing.T) {
	err := osinstaller.DownloadAndExtractIfNeeded(&sastconfiguration.Params)
	assert.NilError(t, err)
}
