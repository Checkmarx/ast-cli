package credentialstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/checkmarx/ast-cli/internal/configfile"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

const (
	checkmarxDirName  = ".checkmarx"
	checkmarxFileName = "checkmarxcli.yaml"
)

// Resolver resolves credential values across explicit, env, keyring and config-file layers.
type Resolver struct {
	mu            sync.Mutex
	canonicalPath string
	filePath      string
	policy        Policy
	store         CredentialStore
	explicit      map[string]string
	migrationOnce sync.Once
}

// NewResolver builds a resolver for configFilePath; a nil store defaults to the keyring.
func NewResolver(configFilePath string, policy Policy, store CredentialStore) *Resolver {
	canonical := CanonicalConfigPath(configFilePath)
	if store == nil {
		store = NewCredentialStore(canonical)
	}
	return &Resolver{
		canonicalPath: canonical,
		filePath:      configFilePath,
		policy:        policy,
		store:         store,
		explicit:      make(map[string]string),
	}
}

// SetExplicit registers an in-process override for a credential.
// The cobra layer resets overrides before every command via ResetExplicit.
func (r *Resolver) SetExplicit(credentialName, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.explicit[credentialName] = value
}

// ResetExplicit discards every in-process explicit override.
func (r *Resolver) ResetExplicit() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.explicit = make(map[string]string)
}

// migrateLazily runs migration on first credential access instead of at
// command startup, so credential-free commands never touch the keyring.
func (r *Resolver) migrateLazily(ctx context.Context) {
	r.migrationOnce.Do(func() {
		r.RunMigration(ctx)
	})
}

// Resolve returns the credential value following the configured policy
// precedence, registering non-empty results with the logger's sanitizer.
func (r *Resolver) Resolve(ctx context.Context, credentialName string) (string, error) {
	if !IsValidCredentialName(credentialName) {
		return "", ErrInvalidName
	}
	r.migrateLazily(ctx)
	value, err := r.resolve(ctx, credentialName)
	if err == nil && value != "" {
		logger.RegisterSensitiveValue(value)
	}
	return value, err
}

// Store persists a credential following the policy: the OS keyring by
// default, the config file under PolicyDisabled.
func (r *Resolver) Store(ctx context.Context, credentialName, value string) error {
	if !IsValidCredentialName(credentialName) {
		return ErrInvalidName
	}
	r.migrateLazily(ctx)
	if r.policy == PolicyDisabled {
		return configfile.SetKey(r.filePath, viperKeyFor(credentialName), value)
	}
	return r.store.Set(ctx, credentialName, value)
}

// Clear removes a credential following the policy, mirroring Store. A missing
// credential is reported as ErrNotFound.
func (r *Resolver) Clear(ctx context.Context, credentialName string) error {
	if !IsValidCredentialName(credentialName) {
		return ErrInvalidName
	}
	r.migrateLazily(ctx)
	if r.policy == PolicyDisabled {
		config, err := configfile.Load(r.filePath)
		if err != nil {
			return err
		}
		if stringValue(config[viperKeyFor(credentialName)]) == "" {
			return ErrNotFound
		}
		return configfile.RemoveKey(r.filePath, viperKeyFor(credentialName))
	}
	return r.store.Delete(ctx, credentialName)
}

// StoresInConfigFile reports whether this policy persists credentials in the
// config file (PolicyDisabled) instead of the OS keyring.
func (r *Resolver) StoresInConfigFile() bool {
	return r.policy == PolicyDisabled
}

// The explicit layer distinguishes "flag passed" from "flag absent" by map
// presence, so --apikey "" still wins over env/keyring/config for this call.
func (r *Resolver) resolve(ctx context.Context, credentialName string) (string, error) {
	if value, ok := r.explicitValue(credentialName); ok {
		return value, nil
	}
	if value := envValue(credentialName); value != "" {
		return value, nil
	}
	switch r.policy {
	case PolicyDisabled:
		return r.resolveFromConfigFile(credentialName)
	case PolicyRequired:
		return r.store.Get(ctx, credentialName)
	default:
		return r.resolveAuto(ctx, credentialName)
	}
}

func (r *Resolver) explicitValue(credentialName string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.explicit[credentialName]
	return value, ok
}

func (r *Resolver) resolveFromConfigFile(credentialName string) (string, error) {
	config, err := configfile.Load(r.filePath)
	if err != nil {
		return "", err
	}
	value := stringValue(config[viperKeyFor(credentialName)])
	if value == "" {
		return "", ErrNotFound
	}
	return value, nil
}

func (r *Resolver) resolveAuto(ctx context.Context, credentialName string) (string, error) {
	current, err := r.store.Get(ctx, credentialName)
	if err == nil {
		return current, nil
	}
	// An unreachable keyring degrades to the config file like a missing entry;
	// access-denied stays loud since a locked keychain needs the user.
	if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrKeyringUnavailable) {
		return "", err
	}
	value, cfgErr := r.resolveFromConfigFile(credentialName)
	if cfgErr == nil {
		if errors.Is(err, ErrKeyringUnavailable) {
			logger.PrintIfVerbose("credentialstore: keyring unavailable, using config-file credential")
		}
		return value, nil
	}
	if errors.Is(err, ErrKeyringUnavailable) {
		return "", fmt.Errorf("%w; set CX_APIKEY or CX_KEYRING_MODE=disabled to keep using the config file", err)
	}
	return "", err
}

// RemoveConfigFileEntry removes the plaintext entry for credentialName from the config file.
func (r *Resolver) RemoveConfigFileEntry(credentialName string) error {
	return configfile.RemoveKey(r.filePath, viperKeyFor(credentialName))
}

var (
	defaultResolver *Resolver
	resolverOnce    sync.Once
)

// Default returns the process-wide resolver built from the active CLI configuration.
func Default() *Resolver {
	resolverOnce.Do(func() {
		path := viper.GetString(params.ConfigFilePathKey)
		if path == "" {
			// Read directly so isolation (and any consumer without viper
			// bindings) still honors the environment variable.
			path = os.Getenv(params.ConfigFilePathEnv)
		}
		if path == "" {
			path = filepath.Join(homeDir(), checkmarxDirName, checkmarxFileName)
		}
		defaultResolver = NewResolver(path, PolicyFromEnv(), nil)
	})
	return defaultResolver
}

// Resolve looks up a credential using the default resolver.
func Resolve(credentialName string) (string, error) {
	return Default().Resolve(context.Background(), credentialName)
}

// SetExplicitCredential registers an override on the default resolver.
func SetExplicitCredential(credentialName, value string) {
	Default().SetExplicit(credentialName, value)
}

// ResetExplicitCredentials clears every override on the default resolver.
func ResetExplicitCredentials() {
	Default().ResetExplicit()
}

// stringValue renders a decoded YAML value as its plaintext form.
func stringValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func homeDir() string {
	if currentUser, err := user.Current(); err == nil && currentUser.HomeDir != "" {
		return currentUser.HomeDir
	}
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		return "."
	}
	return dir
}
