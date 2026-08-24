package credentialstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/stretchr/testify/assert"
)

type fakeStore struct {
	mu       sync.Mutex
	values   map[string]string
	getCalls int
	getErr   error
	setErr   error
}

func newFakeStore() *fakeStore {
	return &fakeStore{values: make(map[string]string)}
}

func (f *fakeStore) Get(_ context.Context, credentialName string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	if f.getErr != nil {
		return "", f.getErr
	}
	value, ok := f.values[credentialName]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}

func (f *fakeStore) Set(_ context.Context, credentialName, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	f.values[credentialName] = value
	return nil
}

func (f *fakeStore) Delete(_ context.Context, credentialName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.values, credentialName)
	return nil
}

func (f *fakeStore) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getCalls
}

var errBackendDown = errors.New("dbus: failed to connect to socket")

func writePlaintextConfig(t *testing.T, path, content string) {
	t.Helper()
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

const yamlWithAPIKey = "cx_apikey: yaml-secret\n"

func TestResolvePrecedence(t *testing.T) {
	cases := []struct {
		name     string
		policy   Policy
		explicit string
		envValue string
		storeVal string
		storeErr error
		yaml     string
		want     string
		wantErr  error
	}{
		{name: "explicit wins over everything", policy: PolicyAuto, explicit: "expl", envValue: "env", storeVal: "store", yaml: yamlWithAPIKey, want: "expl"},
		{name: "env beats store", policy: PolicyAuto, envValue: "env", storeVal: "store", yaml: yamlWithAPIKey, want: "env"},
		{name: "store beats yaml", policy: PolicyAuto, storeVal: "store", yaml: yamlWithAPIKey, want: "store"},
		{name: "auto falls back to yaml on store miss", policy: PolicyAuto, yaml: yamlWithAPIKey, want: "yaml-secret"},
		{name: "auto empty env ignored", policy: PolicyAuto, envValue: "", storeVal: "store", want: "store"},
		{name: "required store hit", policy: PolicyRequired, storeVal: "store", yaml: yamlWithAPIKey, want: "store"},
		{name: "required ignores yaml on miss", policy: PolicyRequired, yaml: yamlWithAPIKey, wantErr: ErrNotFound},
		{name: "required backend error propagates", policy: PolicyRequired, storeErr: errBackendDown, wantErr: errBackendDown},
		{name: "auto backend error propagates without yaml fallback", policy: PolicyAuto, storeErr: errBackendDown, yaml: yamlWithAPIKey, wantErr: errBackendDown},
		{name: "disabled uses yaml", policy: PolicyDisabled, storeVal: "store", yaml: yamlWithAPIKey, want: "yaml-secret"},
		{name: "disabled env used", policy: PolicyDisabled, envValue: "env", yaml: yamlWithAPIKey, want: "env"},
		{name: "disabled nothing anywhere", policy: PolicyDisabled, wantErr: ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
			if tc.yaml != "" {
				writePlaintextConfig(t, configPath, tc.yaml)
			}
			if tc.envValue != "" || (tc.name == "auto empty env ignored") {
				t.Setenv(params.AstAPIKeyEnv, tc.envValue)
			}
			store := newFakeStore()
			if tc.storeVal != "" {
				store.values[CredentialAPIKey] = tc.storeVal
			}
			store.getErr = tc.storeErr

			resolver := NewResolver(configPath, tc.policy, store)
			if tc.explicit != "" {
				resolver.SetExplicit(CredentialAPIKey, tc.explicit)
			}

			got, err := resolver.Resolve(context.Background(), CredentialAPIKey)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestExplicitEmptyOverridesEnvAndStore pins the fix for --apikey "" (a
// flag the user actually typed) losing to CX_APIKEY/keyring: an explicitly
// set credential, even empty, must win over every lower layer for this
// invocation, matching pre-keyring viper flag-over-env precedence.
func TestExplicitEmptyOverridesEnvAndStore(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, yamlWithAPIKey)
	t.Setenv(params.AstAPIKeyEnv, "env-secret")
	store := newFakeStore()
	store.values[CredentialAPIKey] = "store-secret"

	resolver := NewResolver(configPath, PolicyAuto, store)
	resolver.SetExplicit(CredentialAPIKey, "")

	got, err := resolver.Resolve(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "", got)
}

// SetExplicitCredential is the package-level wrapper over
// Default().SetExplicit, exercised through the same injected-resolver seam
// TestMigratePackageLevelWrapper uses.
func TestSetExplicitCredentialPackageLevelWrapper(t *testing.T) {
	ResetForTest()
	defer ResetForTest()

	store := newFakeStore()
	SetDefaultResolverForTest(NewResolver(filepath.Join(t.TempDir(), "checkmarxcli.yaml"), PolicyAuto, store))

	SetExplicitCredential(CredentialAPIKey, "explicit-secret")

	got, err := Resolve(CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "explicit-secret", got)
}

// An unreachable keyring degrades to the config-file layer instead of failing
// the command outright.
func TestResolveAutoKeyringUnavailableFallsBackToYAML(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, yamlWithAPIKey)
	store := newFakeStore()
	store.getErr = ErrKeyringUnavailable
	resolver := NewResolver(configPath, PolicyAuto, store)

	value, err := resolver.Resolve(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "yaml-secret", value)
}

// When the keyring is unreachable and the config file has nothing either, the
// error must name the escape hatches rather than a bare "not found".
func TestResolveAutoKeyringUnavailableAndConfigMissingNamesEscapeHatch(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	store := newFakeStore()
	store.getErr = ErrKeyringUnavailable
	resolver := NewResolver(configPath, PolicyAuto, store)

	_, err := resolver.Resolve(context.Background(), CredentialAPIKey)
	assert.ErrorIs(t, err, ErrKeyringUnavailable)
	assert.ErrorContains(t, err, "CX_APIKEY")
	assert.ErrorContains(t, err, "CX_KEYRING_MODE")
}

func TestResolveDisabledNeverConsultsStore(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, yamlWithAPIKey)
	store := newFakeStore()
	resolver := NewResolver(configPath, PolicyDisabled, store)

	got, err := resolver.Resolve(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "yaml-secret", got)
	assert.Equal(t, 0, store.calls())
}

func TestResolveInvalidName(t *testing.T) {
	resolver := NewResolver(filepath.Join(t.TempDir(), "checkmarxcli.yaml"), PolicyAuto, newFakeStore())
	_, err := resolver.Resolve(context.Background(), "nope")
	assert.ErrorIs(t, err, ErrInvalidName)
}

func TestDefaultResolverSingletonAndReset(t *testing.T) {
	ResetForTest()
	defer ResetForTest()

	first := Default()
	second := Default()
	assert.Same(t, first, second)
	assert.NotEmpty(t, first.filePath)
	assert.NotNil(t, Default())
}
