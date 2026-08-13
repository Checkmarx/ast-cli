package mock

// CredentialStoreMock is an in-memory CredentialStore for unit tests.
type CredentialStoreMock struct {
	Store map[string]string
}

// NewCredentialStoreMock returns an empty in-memory credential store.
func NewCredentialStoreMock() *CredentialStoreMock {
	return &CredentialStoreMock{Store: map[string]string{}}
}

// GetSecret retrieves a secret value from the in-memory store.
func (m *CredentialStoreMock) GetSecret(key string) (string, error) {
	if m.Store == nil {
		return "", nil
	}
	return m.Store[key], nil
}

// SetSecret stores a secret value in the in-memory store.
func (m *CredentialStoreMock) SetSecret(key, value string) error {
	if m.Store == nil {
		m.Store = map[string]string{}
	}
	m.Store[key] = value
	return nil
}

// DeleteSecret removes a secret value from the in-memory store.
func (m *CredentialStoreMock) DeleteSecret(key string) error {
	delete(m.Store, key)
	return nil
}
