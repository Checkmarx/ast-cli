package credentialstore

import (
	"context"
	"errors"

	"github.com/checkmarx/ast-cli/internal/configfile"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
)

// RunMigration moves plaintext config-file credentials into the keyring and removes them from
// the file. It exits cheaply when the active config file holds neither secret,
// so per-invocation calls are a single file read on the steady state.
func (r *Resolver) RunMigration(ctx context.Context) {
	if r.policy == PolicyDisabled {
		return
	}
	config, err := configfile.Load(r.filePath)
	if err != nil || len(config) == 0 {
		return
	}
	for _, name := range []string{CredentialAPIKey, CredentialClientSecret} {
		if stringValue(config[viperKeyFor(name)]) != "" {
			r.migrateOne(ctx, name, config)
		}
	}
}

// Migrate runs migration on the default resolver.
func Migrate() {
	Default().RunMigration(context.Background())
}

func (r *Resolver) migrateOne(ctx context.Context, credentialName string, config map[string]interface{}) {
	yamlValue := stringValue(config[viperKeyFor(credentialName)])
	if yamlValue == "" {
		return
	}
	current, err := r.store.Get(ctx, credentialName)
	switch {
	case errors.Is(err, ErrNotFound):
		r.migrateAndRemove(ctx, credentialName, yamlValue)
	case err != nil:
		logger.PrintfIfVerbose("credentialstore: skipping migration of %s, keyring unavailable: %v", credentialName, err)
	case current == yamlValue:
		r.removeConfigFileEntryQuietly(credentialName)
	default:
		logger.PrintfIfVerbose(
			"credentialstore: keyring already holds a different %s; keeping existing config file entry",
			credentialName,
		)
	}
}

func (r *Resolver) migrateAndRemove(ctx context.Context, credentialName, value string) {
	if err := r.store.Set(ctx, credentialName, value); err != nil {
		logger.PrintfIfVerbose("credentialstore: could not store %s in keyring, keeping config file entry: %v", credentialName, err)
		return
	}
	stored, err := r.store.Get(ctx, credentialName)
	if err != nil || stored != value {
		logger.PrintfIfVerbose("credentialstore: verification of %s failed, keeping config file entry", credentialName)
		return
	}
	r.removeConfigFileEntryQuietly(credentialName)
	logger.PrintfIfVerbose("credentialstore: migrated %s to OS keyring", credentialName)
}

func (r *Resolver) removeConfigFileEntryQuietly(credentialName string) {
	if err := configfile.RemoveKey(r.filePath, viperKeyFor(credentialName)); err != nil {
		logger.PrintfIfVerbose("credentialstore: could not remove %s from config file: %v", credentialName, err)
	}
}

func viperKeyFor(credentialName string) string {
	switch credentialName {
	case CredentialAPIKey:
		return params.AstAPIKey
	case CredentialClientSecret:
		return params.AccessKeySecretConfigKey
	default:
		return ""
	}
}
