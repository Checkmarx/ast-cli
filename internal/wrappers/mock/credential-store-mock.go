package mock

import (
	"context"
	"sync"

	"github.com/checkmarx/ast-cli/internal/credentialstore"
)

type CredentialStoreMock struct {
	mu        sync.Mutex
	Store     map[string]string
	GetErr    error
	SetErr    error
	DeleteErr error
}

func NewCredentialStoreMock() *CredentialStoreMock {
	return &CredentialStoreMock{Store: make(map[string]string)}
}

// Get returns the stored value for credentialName, or GetErr/ErrNotFound.
func (m *CredentialStoreMock) Get(_ context.Context, credentialName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetErr != nil {
		return "", m.GetErr
	}
	value, ok := m.Store[credentialName]
	if !ok {
		return "", credentialstore.ErrNotFound
	}
	return value, nil
}

// Set stores value under credentialName, or returns SetErr.
func (m *CredentialStoreMock) Set(_ context.Context, credentialName, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SetErr != nil {
		return m.SetErr
	}
	m.Store[credentialName] = value
	return nil
}

// Delete removes credentialName, or returns DeleteErr/ErrNotFound.
func (m *CredentialStoreMock) Delete(_ context.Context, credentialName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	if _, ok := m.Store[credentialName]; !ok {
		return credentialstore.ErrNotFound
	}
	delete(m.Store, credentialName)
	return nil
}

var _ credentialstore.CredentialStore = (*CredentialStoreMock)(nil)
