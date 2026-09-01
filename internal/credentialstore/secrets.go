package credentialstore

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/params"
)

const (
	// CredentialAPIKey identifies the cx_apikey credential slot, which holds
	// either a classic API key or an auth-login refresh token.
	CredentialAPIKey = "cx_apikey"
	// CredentialClientSecret identifies the cx_client_secret credential slot.
	CredentialClientSecret = "cx_client_secret"
)

// IsValidCredentialName reports whether name is a known logical credential.
func IsValidCredentialName(name string) bool {
	return name == CredentialAPIKey || name == CredentialClientSecret
}

// IsSecret reports whether viperKey maps to a secret config value.
func IsSecret(viperKey string) bool {
	return viperKey == params.AstAPIKey || viperKey == params.AccessKeySecretConfigKey
}

// CredentialForViperKey maps a secret viper key to its logical credential name.
func CredentialForViperKey(viperKey string) (string, bool) {
	switch viperKey {
	case params.AstAPIKey:
		return CredentialAPIKey, true
	case params.AccessKeySecretConfigKey:
		return CredentialClientSecret, true
	default:
		return "", false
	}
}

func envVarFor(credentialName string) (string, bool) {
	switch credentialName {
	case CredentialAPIKey:
		return params.AstAPIKeyEnv, true
	case CredentialClientSecret:
		return params.AccessKeySecretEnv, true
	default:
		return "", false
	}
}

func envValue(credentialName string) string {
	envVar, ok := envVarFor(credentialName)
	if !ok {
		return ""
	}
	return os.Getenv(envVar)
}
