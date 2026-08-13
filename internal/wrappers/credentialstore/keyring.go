package credentialstore

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const keyringService = "checkmarx-cli"

type keyringStore struct{}

// NewKeyringStore returns a CredentialStore backed by the OS secret store
// (Windows Credential Manager / macOS Keychain / Secret Service via D-Bus).
func NewKeyringStore() CredentialStore { return &keyringStore{} }

func (s *keyringStore) GetSecret(key string) (string, error) {
	value, err := keyring.Get(keyringService, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *keyringStore) SetSecret(key, value string) error {
	return keyring.Set(keyringService, key, value)
}

func (s *keyringStore) DeleteSecret(key string) error {
	err := keyring.Delete(keyringService, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
