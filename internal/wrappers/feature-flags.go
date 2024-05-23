package wrappers

import (
	feature_flags "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"github.com/checkmarx/ast-cli/internal/logger"
)

const tenantIDClaimKey = "tenant_id"
const PackageEnforcementEnabled = "PACKAGE_ENFORCEMENT_ENABLED"
const CVSSV3Enabled = "CVSS_V3_ENABLED"
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
	{
		CommandName: "cx triage update",
		FeatureFlags: []FlagBase{
			{
				Name:    CVSSV3Enabled,
				Default: false,
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
	DefaultFFLoad = true
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
