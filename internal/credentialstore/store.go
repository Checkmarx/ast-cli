package credentialstore

import "context"

// CredentialStore persists logical credentials in the OS keyring.
type CredentialStore interface {
	Get(ctx context.Context, credentialName string) (string, error)
	Set(ctx context.Context, credentialName, value string) error
	Delete(ctx context.Context, credentialName string) error
}

type keyringProvider interface {
	Get(ctx context.Context, service, account string) (string, error)
	Set(ctx context.Context, service, account, value string) error
	Delete(ctx context.Context, service, account string) error
}

type keyCredentialStore struct {
	canonicalPath string
	backend       keyringProvider
}

// NewCredentialStore returns a keyring-backed store scoped to canonicalConfigPath.
func NewCredentialStore(canonicalConfigPath string) CredentialStore {
	return &keyCredentialStore{canonicalPath: canonicalConfigPath, backend: osKeyring{}}
}

func (s *keyCredentialStore) Get(ctx context.Context, credentialName string) (string, error) {
	account, err := s.account(credentialName)
	if err != nil {
		return "", err
	}
	return s.backend.Get(ctx, KeyringServiceName, account)
}

func (s *keyCredentialStore) Set(ctx context.Context, credentialName, value string) error {
	account, err := s.account(credentialName)
	if err != nil {
		return err
	}
	return s.backend.Set(ctx, KeyringServiceName, account, value)
}

func (s *keyCredentialStore) Delete(ctx context.Context, credentialName string) error {
	account, err := s.account(credentialName)
	if err != nil {
		return err
	}
	return s.backend.Delete(ctx, KeyringServiceName, account)
}

func (s *keyCredentialStore) account(credentialName string) (string, error) {
	if !IsValidCredentialName(credentialName) {
		return "", ErrInvalidName
	}
	return AccountFor(s.canonicalPath, credentialName), nil
}
