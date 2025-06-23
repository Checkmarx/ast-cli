package wrappers

import (
	"errors"

	"github.com/checkmarx/ast-cli/internal/logger"
)

const CustomStatesFeatureFlag = "CUSTOM_STATES_ENABLED"
const tenantIDClaimKey = "tenant_id"
const PackageEnforcementEnabled = "PACKAGE_ENFORCEMENT_ENABLED"
const CVSSV3Enabled = "CVSS_V3_ENABLED"
const MinioEnabled = "MINIO_ENABLED"
const SastCustomStateEnabled = "SAST_CUSTOM_STATES_ENABLED"
const ContainerEngineCLIEnabled = "CONTAINER_ENGINE_CLI_ENABLED"
const SCSEngineCLIEnabled = "NEW_2MS_SCORECARD_RESULTS_CLI_ENABLED"
const NewScanReportEnabled = "NEW_SAST_SCAN_REPORT_ENABLED"
const RiskManagementEnabled = "RISK_MANAGEMENT_IDES_PROJECT_RESULTS_SCORES_API_ENABLED"
const OssRealtimeEnabled = "OSS_REALTIME_ENABLED"
const maxRetries = 3

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
		CommandName: "cx triage get-states",
		FeatureFlags: []FlagBase{
			{
				Name:    CustomStatesFeatureFlag,
				Default: false,
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
	{
		CommandName: "cx triage update",
		FeatureFlags: []FlagBase{
			{
				Name:    CVSSV3Enabled,
				Default: false,
			},
			{
				Name:    SastCustomStateEnabled,
				Default: false,
			},
		},
	},
	{
		CommandName: "cx results risk-management",
		FeatureFlags: []FlagBase{
			{
				Name:    RiskManagementEnabled,
				Default: false,
			},
		},
	},
}

var featureFlags = map[string]bool{}
var featureFlagsCache = map[string]bool{}

func HandleFeatureFlags(featureFlagsWrapper FeatureFlagsWrapper) error {
	allFlags, err := featureFlagsWrapper.GetAll()
	if err != nil {
		LoadFeatureFlagsDefaultValues()
		return nil
	}

	loadFeatureFlagsMap(*allFlags)
	return nil
}

func GetSpecificFeatureFlag(featureFlagsWrapper FeatureFlagsWrapper, flagName string) (*FeatureFlagResponseModel, error) {
	if value, exists := featureFlagsCache[flagName]; exists {
		return &FeatureFlagResponseModel{Name: flagName, Status: value}, nil
	}

	specificFlag, err := getSpecificFlagWithRetry(featureFlagsWrapper, flagName, maxRetries)
	if err != nil {
		if len(featureFlags) == 0 || DefaultFFLoad {
			_ = HandleFeatureFlags(featureFlagsWrapper)
		}
		// Take the value from FeatureFlags
		return &FeatureFlagResponseModel{Name: flagName, Status: featureFlags[flagName]}, nil
	}

	UpdateSpecificFeatureFlagMap(flagName, *specificFlag)
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
		logger.PrintfIfVerbose("Retry %d/%d for flag %s failed with error: %v", i+1, retries, flagName, err)
	}

	logger.PrintfIfVerbose("Failed to get feature flag %s after %d retries", flagName, retries)
	return nil, errors.New("failed to get feature flag after retries")
}

func UpdateSpecificFeatureFlagMap(flagName string, flag FeatureFlagResponseModel) {
	featureFlagsCache[flagName] = flag.Status
}

func ClearCache() {
	featureFlagsCache = map[string]bool{}
}

func loadFeatureFlagsMap(allFlags FeatureFlagsResponseModel) {
	for _, flag := range allFlags {
		featureFlags[flag.Name] = flag.Status
	}

	// Update FeatureFlags map with default values in case it does not exist in all flags response
	for _, cmdFlag := range FeatureFlagsBaseMap {
		for _, flag := range cmdFlag.FeatureFlags {
			_, ok := featureFlags[flag.Name]
			if !ok {
				featureFlags[flag.Name] = flag.Default
			}
		}
	}
	DefaultFFLoad = false
}

func LoadFeatureFlagsDefaultValues() {
	logger.PrintIfVerbose("Get feature flags failed. Loading defaults...")

	for _, cmdFlag := range FeatureFlagsBaseMap {
		for _, flag := range cmdFlag.FeatureFlags {
			featureFlags[flag.Name] = flag.Default
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
