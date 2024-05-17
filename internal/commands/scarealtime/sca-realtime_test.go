package scarealtime

import (
	"io/ioutil"
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

	// Ensure we have results to read
	err = copyResultsToTempDir()
	assert.NilError(t, err)

	err = GetSCAVulnerabilities(mock.ScaRealTimeHTTPMockWrapper{})
	assert.NilError(t, err)

	// Run second time to cover SCA Resolver download not needed code
	err = cmd.Execute()
	assert.NilError(t, err)
}

func copyResultsToTempDir() error {
	// Read all content of src to data, may cause OOM for a large file.
	data, err := ioutil.ReadFile("../data/cx-sca-realtime-results.json")
	if err != nil {
		return err
	}
	// Write data to dst
	err = ioutil.WriteFile(ScaResolverResultsFileNameDir, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func TestRequiredProjectDir(t *testing.T) {
	invalidProjectPath := "/invalid/project/dir"
	args := []string{"scan", "sca-realtime", "--project-dir", invalidProjectPath}
	cmd := NewScaRealtimeCommand(mock.ScaRealTimeHTTPMockWrapper{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	assert.Error(t, err, "Provided path does not exist: "+invalidProjectPath, err.Error())
}
