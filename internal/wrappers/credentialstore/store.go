// Package credentialstore persists the CLI's secrets in the OS keyring, with
// yaml-config fallback when no keyring is available.
package credentialstore

import "github.com/checkmarx/ast-cli/internal/wrappers/configuration"

// CredentialStore reads/writes/deletes a secret by viper key; Get returns "" when absent.
type CredentialStore interface {
	GetSecret(key string) (string, error)
	SetSecret(key, value string) error
	DeleteSecret(key string) error
}

// Default is the process-wide store; file-backed until main installs the keyring chain.
var Default CredentialStore = NewFileStore()

// Install wires the store into the package default and the configuration secret router.
func Install(s CredentialStore) {
	Default = s
	configuration.Secrets = s
}
