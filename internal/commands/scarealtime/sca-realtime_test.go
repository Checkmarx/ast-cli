package scarealtime

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

var (
	_, b, _, _       = runtime.Caller(0)
	projectDirectory = filepath.Dir(b)
)

func TestRunScaRealtime(t *testing.T) {
	args := []string{"scan", "sca-realtime", "--project-dir", projectDirectory}
	cmd := NewScaRealtimeCommand(mock.ScaRealTimeHTTPMockWrapper{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	assert.NilError(t, err)
}

func TestRequiredProjectDir(t *testing.T) {
	invalidProjectPath := "/invalid/project/dir"
	args := []string{"scan", "sca-realtime", "--project-dir", invalidProjectPath}
	cmd := NewScaRealtimeCommand(mock.ScaRealTimeHTTPMockWrapper{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	assert.Error(t, err, "Provided path does not exist: "+invalidProjectPath, err.Error())
}
