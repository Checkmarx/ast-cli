package wrappers

import (
	"github.com/checkmarx/ast-cli/internal/logger"
)

const tenantIDClaimKey = "tenant_id"
const PackageEnforcementEnabled = "PACKAGE_ENFORCEMENT_ENABLED"
const MinioEnabled = "MINIO_ENABLED"
const ContainerEngineCLIEnabled = "CONTAINER_ENGINE_CLI_ENABLED"

var FeatureFlagsBaseMap = []CommandFlags{
	{
		CommandName: "cx scan create",
		FeatureFlags: []FlagBase{
			{
				Name:    PackageEnforcementEnabled,
				Default: true,
			},
			{
				Name:    MinioEnabled,
				Default: true,
			},
		},
	},
	{
		CommandName: "cx project create",
	},
	{
		CommandName: "cx import",
		FeatureFlags: []FlagBase{
			{
				Name:    MinioEnabled,
				Default: true,
			},
		},
	},
}

var FeatureFlags = map[string]bool{}

func HandleFeatureFlags(featureFlagsWrapper FeatureFlagsWrapper) error {
	allFlags, err := featureFlagsWrapper.GetAll()
	if err != nil {
		loadFeatureFlagsDefaultValues()

		return nil
	}

	loadFeatureFlagsMap(*allFlags)

	return nil
}

func loadFeatureFlagsMap(allFlags FeatureFlagsResponseModel) {
	for _, flag := range allFlags {
		FeatureFlags[flag.Name] = flag.Status
	}

	//  Update FeatureFlags map with default values in case it does not exist in all flags response
	for _, cmdFlag := range FeatureFlagsBaseMap {
		for _, flag := range cmdFlag.FeatureFlags {
			_, ok := FeatureFlags[flag.Name]
			if !ok {
				FeatureFlags[flag.Name] = flag.Default
			}
		}
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
