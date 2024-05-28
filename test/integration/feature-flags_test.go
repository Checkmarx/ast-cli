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

func TestHandleFeatureFlagsIntegration(t *testing.T) {

	createASTIntegrationTestCommand(t)
	featureFlagsPath := viper.GetString(commonParams.FeatureFlagsKey)
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)

	err := wrappers.HandleFeatureFlags(featureFlagsWrapper)
	assert.NilError(t, err, "HandleFeatureFlags should not return an error")
	assert.Assert(t, len(wrappers.FeatureFlags) > 0, "FeatureFlags map should not be empty")
}

func TestGetSpecificFeatureFlagIntegration(t *testing.T) {
	createASTIntegrationTestCommand(t)
	featureFlagsPath := viper.GetString(commonParams.FeatureFlagsKey)
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)

	flagName := featureFlagsConstants.ByorEnabled
	flagResponse, err := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, flagName)
	assert.NilError(t, err, "GetSpecificFeatureFlag should not return an error")
	assert.Equal(t, flagResponse.Name, flagName, "Feature flag name should match")
}

func TestGetSpecificFeatureFlagWithRetryIntegration(t *testing.T) {
	createASTIntegrationTestCommand(t)
	featureFlagsPath := viper.GetString(commonParams.FeatureFlagsKey)
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)

	flagName := featureFlagsConstants.ByorEnabled
	flagResponse, err := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, flagName)
	assert.NilError(t, err, "GetSpecificFeatureFlagWithRetry should not return an error")
	assert.Equal(t, flagResponse.Name, flagName, "Feature flag name should match")
}

func TestUpdateSpecificFeatureFlagMapIntegration(t *testing.T) {
	flagName := featureFlagsConstants.ByorEnabled
	flag := wrappers.FeatureFlagResponseModel{Name: flagName, Status: true}
	wrappers.UpdateSpecificFeatureFlagMap(flagName, flag)
	assert.Equal(t, wrappers.FeatureFlagsSpecific[flagName], flag.Status, "Feature flag status should be updated")
}

func TestUpdateSpecificFeatureFlagMapWithDefaultIntegration(t *testing.T) {
	flagName := featureFlagsConstants.ByorEnabled

	// Ensure the default is set to false for testing purposes
	wrappers.FeatureFlagsBaseMap = []wrappers.CommandFlags{
		{
			CommandName: "cx scan create",
			FeatureFlags: []wrappers.FlagBase{
				{Name: flagName, Default: true},
			},
		},
	}

	wrappers.UpdateSpecificFeatureFlagMapWithDefault(flagName)
	assert.Equal(t, true, wrappers.FeatureFlagsSpecific[flagName], "Feature flag status should be updated to default")
}

func TestLoadFeatureFlagsDefaultValuesIntegration(t *testing.T) {
	wrappers.LoadFeatureFlagsDefaultValues()
	assert.Assert(t, len(wrappers.FeatureFlags) > 0, "FeatureFlags map should not be empty after loading defaults")
}
