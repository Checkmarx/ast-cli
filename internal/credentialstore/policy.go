package credentialstore

import (
	"fmt"
	"os"

	"github.com/checkmarx/ast-cli/internal/logger"
)

// Policy controls how the resolver combines keyring, environment and YAML layers.
type Policy int

const (
	// PolicyAuto prefers the keyring and falls back to the config-file layer.
	PolicyAuto Policy = iota
	// PolicyRequired uses only the keyring, never YAML.
	PolicyRequired
	// PolicyDisabled ignores the keyring entirely.
	PolicyDisabled
)

// KeyringModeEnvVar selects the keyring policy via environment variable.
const KeyringModeEnvVar = "CX_KEYRING_MODE"

const (
	policyValueAuto     = "auto"
	policyValueRequired = "required"
	policyValueDisabled = "disabled"
)

// ParsePolicy converts a raw mode string into a Policy.
func ParsePolicy(raw string) (Policy, error) {
	switch raw {
	case "", policyValueAuto:
		return PolicyAuto, nil
	case policyValueRequired:
		return PolicyRequired, nil
	case policyValueDisabled:
		return PolicyDisabled, nil
	default:
		return PolicyAuto, fmt.Errorf(
			"invalid keyring mode %q, valid values: %s, %s, %s",
			raw, policyValueAuto, policyValueRequired, policyValueDisabled,
		)
	}
}

// PolicyFromEnv reads KeyringModeEnvVar, falling back to PolicyAuto on invalid values.
func PolicyFromEnv() Policy {
	policy, err := ParsePolicy(os.Getenv(KeyringModeEnvVar))
	if err != nil {
		logger.PrintfIfVerbose("credentialstore: %v, falling back to auto mode", err)
		return PolicyAuto
	}
	return policy
}
