package commands

import (
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/config"
)

const (
	astInstallationPathEnv = "AST_INSTALLATION_PATH"

	// Configurable database environment variables
	dbHostEnv     = "DATABASE_HOST"
	dbPortEnv     = "DATABASE_PORT"
	dbUserEnv     = "DATABASE_USER"
	dbPasswordEnv = "DATABASE_PASSWORD"
	dbInstanceEnv = "DATABASE_INSTANCE"

	// Configurable network environment variables
	traefikPort      = "ENTRYPOINT_PORT"
	privateKeyPath   = "TLS_PRIVATE_KEY_PATH"
	certificatePath  = "TLS_CERTIFICATE_PATH"
	fqdn             = "FQDN"
	externalHostname = "EXTERNAL_HOSTNAME"

	// Configurable logging environment variables
	logLevel             = "LOG_LEVEL"
	logRotationAgeDays   = "LOG_ROTATION_AGE_DAYS"
	logRotationMaxSizeMB = "LOG_ROTATION_MAX_SIZE_MB"
)

func getEnvVarsForCommand(configuration *config.SingleNodeConfiguration, astInstallationPath string) []string {
	return []string{
		envKeyAndValue(astInstallationPathEnv, astInstallationPath),

		envKeyAndValue(dbHostEnv, configuration.Database.Host),
		envKeyAndValue(dbPortEnv, configuration.Database.Port),
		envKeyAndValue(dbUserEnv, configuration.Database.Username),
		envKeyAndValue(dbPasswordEnv, configuration.Database.Password),
		envKeyAndValue(dbInstanceEnv, configuration.Database.Instance),

		envKeyAndValue(traefikPort, configuration.Network.EntrypointPort),
		envKeyAndValue(privateKeyPath, configuration.Network.TLS.PrivateKeyPath),
		envKeyAndValue(certificatePath, configuration.Network.TLS.CertificatePath),
		envKeyAndValue(fqdn, configuration.Network.FullyQualifiedDomainName),
		envKeyAndValue(externalHostname, configuration.Network.ExternalHostname),

		envKeyAndValue(logLevel, configuration.Log.Level),
		envKeyAndValue(logRotationAgeDays, configuration.Log.Rotation.MaxAgeDays),
		envKeyAndValue(logRotationMaxSizeMB, configuration.Log.Rotation.MaxSizeMB),
	}
}

func envKeyAndValue(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}
