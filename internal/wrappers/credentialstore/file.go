package credentialstore

import (
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/pkg/errors"
)

// fileStore is the yaml-file backend.
type fileStore struct{}

// NewFileStore returns a CredentialStore backed by the yaml config file.
func NewFileStore() CredentialStore { return &fileStore{} }

func (s *fileStore) GetSecret(key string) (string, error) {
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		return "", err
	}
	config, err := configuration.LoadConfig(configPath)
	if err != nil {
		return "", err
	}
	if v, ok := config[key].(string); ok {
		return v, nil
	}
	return "", nil
}

func (s *fileStore) SetSecret(key, value string) error {
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		return errors.Wrap(err, "failed to resolve config file path")
	}
	return configuration.SafeWriteSingleConfigKeyString(configPath, key, value)
}

func (s *fileStore) DeleteSecret(key string) error {
	return s.SetSecret(key, "")
}
