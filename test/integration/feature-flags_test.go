//go:build integration

package integration

import (
	"testing"

	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

type MockFeatureFlagsWrapper struct {
	AllFlags     *wrappers.FeatureFlagsResponseModel
	SpecificFlag *wrappers.FeatureFlagResponseModel
	Err          error
}

func Test_HandleFeatureFlags_WhenCalled_ThenNoErrorAndCacheNotEmpty(t *testing.T) {
	createASTIntegrationTestCommand(t)
	featureFlagsPath := viper.GetString(commonParams.FeatureFlagsKey)
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)
	//t.Parallel()
	err := wrappers.HandleFeatureFlags(featureFlagsWrapper)
	assert.NilError(t, err, "HandleFeatureFlags should not return an error")
	assert.Assert(t, len(wrappers.FeatureFlagsCache) > 0, "FeatureFlags cache should not be empty")
}

func TestByorEnabled_Flag_should_be_true(t *testing.T) {
	createASTIntegrationTestCommand(t)
	featureFlagsPath := viper.GetString("featureFlagsPath")
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)

	flagName := featureFlagsConstants.ByorEnabled
	t.Parallel()

	flagResponse, err := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, flagName)
	assert.NilError(t, err, "GetSpecificFeatureFlag should not return an error")
	assert.Equal(t, flagResponse.Status, true, "ByorEnabled Feature flag status should be true")

}

func Test_UpdateSpecificFeatureFlagMap_WhenCalled_ThenUpdateCache(t *testing.T) {
	//t.Parallel()
	flagName := featureFlagsConstants.ByorEnabled
	wrappers.FeatureFlagsCache[flagName] = false
	t.Parallel()

	flag := wrappers.FeatureFlagResponseModel{Name: flagName, Status: true}
	wrappers.UpdateSpecificFeatureFlagMap(flagName, flag)
	assert.Equal(t, wrappers.FeatureFlagsCache[flagName], flag.Status, "Feature flag status should be updated")
}

func Test_LoadFeatureFlagsDefaultValues_WhenCalled_ThenFeatureFlagsNotEmpty(t *testing.T) {
	//t.Parallel()
	wrappers.LoadFeatureFlagsDefaultValues()
	assert.Assert(t, len(wrappers.FeatureFlags) > 0, "FeatureFlags cache should not be empty after loading defaults")
}
