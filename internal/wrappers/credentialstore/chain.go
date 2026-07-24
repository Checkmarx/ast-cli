package credentialstore

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/logger"
)

// chainStore tries the primary (keyring) and falls back to the secondary (yaml).
type chainStore struct {
	primary  CredentialStore
	fallback CredentialStore
}

// NewChainStore returns a CredentialStore that tries primary, then fallback.
func NewChainStore(primary, fallback CredentialStore) CredentialStore {
	return &chainStore{primary: primary, fallback: fallback}
}

func (s *chainStore) GetSecret(key string) (string, error) {
	value, err := s.primary.GetSecret(key)
	if err == nil && value != "" {
		return value, nil
	}
	if err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("keyring read failed, using fallback: %v", err))
	}
	return s.fallback.GetSecret(key)
}

func (s *chainStore) SetSecret(key, value string) error {
	if err := s.primary.SetSecret(key, value); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("keyring write failed, using fallback: %v", err))
		return s.fallback.SetSecret(key, value)
	}
	// Keyring write succeeded: scrub any plaintext copy left in yaml.
	if err := s.fallback.DeleteSecret(key); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("failed to scrub yaml secret copy: %v", err))
	}
	return nil
}

// DeleteSecret clears both backends.
func (s *chainStore) DeleteSecret(key string) error {
	if err := s.primary.DeleteSecret(key); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("keyring delete failed (continuing): %v", err))
	}
	return s.fallback.DeleteSecret(key)
}
