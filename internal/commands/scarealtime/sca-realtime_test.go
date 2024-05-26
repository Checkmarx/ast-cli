package scarealtime

import (
	"io/ioutil"
	"os"
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
	err := os.RemoveAll(Params.WorkingDir())
	args := []string{"scan", "sca-realtime", "--project-dir", projectDirectory}
	cmd := NewScaRealtimeCommand(mock.ScaRealTimeHTTPMockWrapper{})
	cmd.SetArgs(args)
	err = cmd.Execute()
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
	scaResolverResultsFileNameDir := filepath.Join(Params.WorkingDir(), ScaResolverResultsFileName)
	err = ioutil.WriteFile(scaResolverResultsFileNameDir, data, 0644)
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

func TestCreateDependencyMapFromDependencyResolution_NugetDependencies_Success(t *testing.T) {
	dependecyResolutionResult := DependencyResolution{
		Dependencies: []Dependency{
			NewDependency("8ce2d33f-5783-4fe1-b9a7-3ce2c9a3aae9", "Microsoft. NETCore. Platforms",
				"1.1.0", "Nuget", []interface{}{"NetStandard20"}),
			NewDependency("60b40261-18b2-4cf6-bdf5-e23ad408de3b", "NETStandard.Library",
				"2.0.3", "Nuget", []interface{}{"NetStandard20"}),
		},
	}
	dependencyMap := createDependencyMapFromDependencyResolution(&dependecyResolutionResult)
	assert.Equal(t, len(dependencyMap), 2)
	assert.Equal(t, dependencyMap["60b40261-18b2-4cf6-bdf5-e23ad408de3b"].PackageManager, "Nuget")
	assert.Equal(t, dependencyMap["60b40261-18b2-4cf6-bdf5-e23ad408de3b"].Version, "2.0.3")
	assert.Equal(t, dependencyMap["60b40261-18b2-4cf6-bdf5-e23ad408de3b"].PackageName, "NETStandard.Library")
	assert.Equal(t, dependencyMap["8ce2d33f-5783-4fe1-b9a7-3ce2c9a3aae9"].PackageManager, "Nuget")
	assert.Equal(t, dependencyMap["8ce2d33f-5783-4fe1-b9a7-3ce2c9a3aae9"].Version, "1.1.0")
	assert.Equal(t, dependencyMap["8ce2d33f-5783-4fe1-b9a7-3ce2c9a3aae9"].PackageName, "Microsoft. NETCore. Platforms")
}

func NewDependency(nodeID, name, version, resolvingModuleType string, targetFrameworks []interface{}) Dependency {
	return Dependency{
		ID:                  NewID(nodeID, name, version),
		ResolvingModuleType: resolvingModuleType,
		TargetFrameworks:    targetFrameworks,
	}
}

func NewID(nodeID, name, version string) ID {
	return ID{
		NodeID:  nodeID,
		Name:    name,
		Version: version,
	}
}
