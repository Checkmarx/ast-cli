package scarealtime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestRunScaRealtimeWrongURLDownload(t *testing.T) {
	err := os.RemoveAll(ScaResolverWorkingDir)
	if err != nil {
		return
	}
	args := []string{"scan", "sca-realtime", "--project-dir", projectDirectory}
	Params.SCAResolverDownloadURL = "http://www.invalid-sca-resolver.com"
	cmd := NewScaRealtimeCommand(mock.ScaRealTimeHTTPMockWrapper{})
	cmd.SetArgs(args)
	err = cmd.Execute()
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(strings.ToLower(err.Error()), strings.ToLower("Invoking HTTP request to upload file failed")))
}

func TestRequiredProjectDir(t *testing.T) {
	invalidProjectPath := "/invalid/project/dir"
	args := []string{"scan", "sca-realtime", "--project-dir", invalidProjectPath}
	cmd := NewScaRealtimeCommand(mock.ScaRealTimeHTTPMockWrapper{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	assert.Error(t, err, "Provided path does not exist: "+invalidProjectPath, err.Error())
}
