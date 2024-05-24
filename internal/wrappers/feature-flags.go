package wrappers

import (
	"errors"
	feature_flags "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"github.com/checkmarx/ast-cli/internal/logger"
)

const tenantIDClaimKey = "tenant_id"
const PackageEnforcementEnabled = "PACKAGE_ENFORCEMENT_ENABLED"
const MinioEnabled = "MINIO_ENABLED"
const ContainerEngineCLIEnabled = "CONTAINER_ENGINE_CLI_ENABLED"
const NewScanReportEnabled = "NEW_SAST_SCAN_REPORT_ENABLED"

var DefaultFFLoad bool = false

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
			{
				Name:    feature_flags.ByorEnabled,
				Default: false,
			},
		},
	},
	{
		CommandName: "cx results show",
		FeatureFlags: []FlagBase{
			{
				Name:    NewScanReportEnabled,
				Default: false,
			},
		},
	},
}

var FeatureFlags = map[string]bool{}
var FeatureFlagsSpecific = map[string]bool{}

func HandleFeatureFlags(featureFlagsWrapper FeatureFlagsWrapper) error {
	allFlags, err := featureFlagsWrapper.GetAll()
	if err != nil {
		loadFeatureFlagsDefaultValues()
		return nil
	}

	loadFeatureFlagsMap(*allFlags)
	return nil
}

func GetSpecificFeatureFlag(featureFlagsWrapper FeatureFlagsWrapper, flagName string) (*FeatureFlagResponseModel, error) {
	if value, exists := FeatureFlagsSpecific[flagName]; exists {
		return &FeatureFlagResponseModel{Name: flagName, Status: value}, nil
	}

	specificFlag, err := getSpecificFlagWithRetry(featureFlagsWrapper, flagName, 5)
	if err != nil {
		updateSpecificFeatureFlagMapWithDefault(flagName)
		return &FeatureFlagResponseModel{Name: flagName, Status: FeatureFlagsSpecific[flagName]}, nil
	}

	updateSpecificFeatureFlagMap(flagName, *specificFlag)
	return specificFlag, nil
}

func getSpecificFlagWithRetry(wrapper FeatureFlagsWrapper, flagName string, retries int) (*FeatureFlagResponseModel, error) {
	var flag *FeatureFlagResponseModel
	var err error

	for i := 0; i < retries; i++ {
		flag, err = wrapper.GetSpecificFlag(flagName)
		if err == nil {
			return flag, nil
		}
	}

	return nil, errors.New("failed to get feature flag after retries")
}

func updateSpecificFeatureFlagMapWithDefault(flagName string) {
	for _, cmdFlag := range FeatureFlagsBaseMap {
		for _, flag := range cmdFlag.FeatureFlags {
			if flag.Name == flagName {
				FeatureFlagsSpecific[flagName] = flag.Default
				FeatureFlags[flagName] = flag.Default
				return
			}
		}
	}
	FeatureFlagsSpecific[flagName] = false // Default to false if not found in base map
	FeatureFlags[flagName] = false         // Ensure FeatureFlags is also updated
}

func updateSpecificFeatureFlagMap(flagName string, flag FeatureFlagResponseModel) {
	FeatureFlagsSpecific[flagName] = flag.Status
	FeatureFlags[flagName] = flag.Status
}

func loadFeatureFlagsMap(allFlags FeatureFlagsResponseModel) {
	for _, flag := range allFlags {
		FeatureFlags[flag.Name] = flag.Status
	}

	// Update FeatureFlags map with default values in case it does not exist in all flags response
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
	DefaultFFLoad = true
}

type FeatureFlagsWrapper interface {
	GetAll() (*FeatureFlagsResponseModel, error)
	GetSpecificFlag(specificFlag string) (*FeatureFlagResponseModel, error)
}

type FeatureFlagsResponseModel []struct {
	Name   string `json:"name"`
	Status bool   `json:"status"`
}
type FeatureFlagResponseModel struct {
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
