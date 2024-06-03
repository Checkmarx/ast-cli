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

	err := wrappers.HandleFeatureFlags(featureFlagsWrapper)
	assert.NilError(t, err, "HandleFeatureFlags should not return an error")
	assert.Assert(t, len(wrappers.FeatureFlagsCache) > 0, "FeatureFlags cache should not be empty")
}

func Test_UpdateSpecificFeatureFlagMap_WhenCalled_ThenUpdateCache(t *testing.T) {
	flagName := featureFlagsConstants.ByorEnabled
	wrappers.FeatureFlagsCache[flagName] = false

	flag := wrappers.FeatureFlagResponseModel{Name: flagName, Status: true}
	wrappers.UpdateSpecificFeatureFlagMap(flagName, flag)
	assert.Equal(t, wrappers.FeatureFlagsCache[flagName], flag.Status, "Feature flag status should be updated")
}

func Test_LoadFeatureFlagsDefaultValues_WhenCalled_ThenFeatureFlagsNotEmpty(t *testing.T) {
	wrappers.LoadFeatureFlagsDefaultValues()
	assert.Assert(t, len(wrappers.FeatureFlags) > 0, "FeatureFlags cache should not be empty after loading defaults")
}

func Test_UpdateSpecificFeatureFlagMapWithDefault_WhenCalled_ThenUpdateCacheToDefault(t *testing.T) {
	flagName := featureFlagsConstants.ByorEnabled
	wrappers.FeatureFlagsCache[flagName] = true

	flag := wrappers.FeatureFlagResponseModel{Name: flagName, Status: false}
	// Ensure the default is set to false for testing purposes
	wrappers.FeatureFlagsBaseMap = []wrappers.CommandFlags{
		{
			CommandName: "cx scan create",
			FeatureFlags: []wrappers.FlagBase{
				{Name: flag.Name, Default: flag.Status},
			},
		},
	}

	wrappers.UpdateSpecificFeatureFlagMapWithDefault(flagName)
	assert.Equal(t, wrappers.FeatureFlagsCache[flagName], flag.Status, "Feature flag status should be updated to default")
}
