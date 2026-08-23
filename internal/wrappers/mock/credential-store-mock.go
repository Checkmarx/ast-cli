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

func (m *CredentialStoreMock) Set(_ context.Context, credentialName string, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SetErr != nil {
		return m.SetErr
	}
	m.Store[credentialName] = value
	return nil
}

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
