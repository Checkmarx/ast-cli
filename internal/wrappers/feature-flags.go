package wrappers

import (
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
)

const tenantIDClaimKey = "tenant_id"
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

func HandleFeatureFlags(featureFlagsWrapper FeatureFlagsWrapper) error {
	accessToken, tokenError := GetAccessToken()
	if tokenError != nil {
		return tokenError
	}

	tenantIDFromClaims, extractError := extractFromTokenClaims(accessToken, tenantIDClaimKey)
	if extractError != nil {
		return extractError
	}

	tenantID := strings.Split(tenantIDFromClaims, "::")[1]
	allFlags, err := featureFlagsWrapper.GetAll(tenantID)
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
	GetAll(tenantID string) (*FeatureFlagsResponseModel, error)
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
