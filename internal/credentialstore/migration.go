package credentialstore

import (
	"context"
	"errors"

	"github.com/checkmarx/ast-cli/internal/configfile"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
)

// RunMigration moves plaintext config-file credentials into the keyring and
// removes them from the file. Called lazily on first credential access.
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
			if r.migrateOne(ctx, name, config) {
				return
			}
		}
	}
}

// Migrate runs migration on the default resolver.
func Migrate() {
	Default().RunMigration(context.Background())
}

// migrateOne migrates a single credential slot; the bool return reports
// keyring unavailability so RunMigration can skip the remaining slots.
func (r *Resolver) migrateOne(ctx context.Context, credentialName string, config map[string]interface{}) bool {
	yamlValue := stringValue(config[viperKeyFor(credentialName)])
	if yamlValue == "" {
		return false
	}
	current, err := r.store.Get(ctx, credentialName)
	switch {
	case errors.Is(err, ErrNotFound):
		r.migrateAndRemove(ctx, credentialName, yamlValue)
	case err != nil:
		logger.PrintfIfVerbose("credentialstore: skipping migration of %s, keyring unavailable: %v", credentialName, err)
		return errors.Is(err, ErrKeyringUnavailable)
	case current == yamlValue:
		r.removeConfigFileEntryQuietly(credentialName)
	default:
		logger.PrintfIfVerbose(
			"credentialstore: keyring already holds a different %s; keeping existing config file entry",
			credentialName,
		)
	}
	return false
}

func (r *Resolver) migrateAndRemove(ctx context.Context, credentialName, value string) {
	if !r.configStillHolds(credentialName, value) {
		logger.PrintfIfVerbose(
			"credentialstore: %s changed in config file during migration; aborting to avoid overwriting a rotated credential",
			credentialName,
		)
		return
	}
	// Re-check the keyring itself: a concurrent `cx auth login` writes there
	// directly, which the config-file recheck above can't detect.
	if _, err := r.store.Get(ctx, credentialName); !errors.Is(err, ErrNotFound) {
		logger.PrintfIfVerbose(
			"credentialstore: %s appeared in keyring during migration; aborting to avoid overwriting a rotated credential",
			credentialName,
		)
		return
	}
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

// configStillHolds reports whether the config file still carries exactly
// value, guarding against a rotation that happened during migration.
func (r *Resolver) configStillHolds(credentialName, value string) bool {
	config, err := configfile.Load(r.filePath)
	if err != nil {
		return false
	}
	return stringValue(config[viperKeyFor(credentialName)]) == value
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
