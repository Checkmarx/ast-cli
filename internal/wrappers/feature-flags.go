package wrappers

import (
	"github.com/checkmarx/ast-cli/internal/logger"
)

const PackageEnforcementEnabled = "PACKAGE_ENFORCEMENT_ENABLED"

var FeatureFlagsBaseMap = []CommandFlags{
	{
		CommandName: "cx scan create",
		FeatureFlags: []FlagBase{
			{
				Name:    PackageEnforcementEnabled,
				Default: true,
			},
		},
	},
}

var FeatureFlags = map[string]bool{}

func HandleFeatureFlags(featureFlagsWrapper FeatureFlagsWrapper) {
	allFlags, err := featureFlagsWrapper.GetAll()

	if err != nil {
		loadFeatureFlagsDefaultValues()
		return
	}

	loadFeatureFlagsMap(*allFlags)
}

func loadFeatureFlagsMap(allFlags FeatureFlagsResponseModel) {
	for _, flag := range allFlags {
		FeatureFlags[flag.Name] = flag.Status
	}
}

func loadFeatureFlagsDefaultValues() {
	logger.PrintIfVerbose("Get feature flags failed. Loading defaults...")

	for _, cmdFlag := range FeatureFlagsBaseMap {
		for _, flag := range cmdFlag.FeatureFlags {
			FeatureFlags[flag.Name] = flag.Default
		}
	}
}

type FeatureFlagsWrapper interface {
	GetAll() (*FeatureFlagsResponseModel, error)
}

type FeatureFlagsResponseModel []struct {
	Name   string `json:"name"`
	Status bool   `json:"status"`
}

type CommandFlags struct {
	CommandName  string
	FeatureFlags []FlagBase
}

type FlagBase struct {
	Name    string
	Default bool
}
