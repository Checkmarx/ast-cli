package commands

import (
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/checkmarxDev/ast-cli/internal/config"
)

const (
	defaultLogLocation     = "/etc/conf/ast"
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
	externalHostname = "EXTERNAL_HOSTNAME"

	// Configurable logging environment variables
	logLevel             = "LOG_LEVEL"
	logLocation          = "LOG_LOCATION"
	logRotationMaxSizeMB = "LOG_ROTATION_MAX_SIZE_MB"
	logRotationCount     = "LOG_ROTATION_COUNT"
)

func createEnvVarsForCommand(configuration *config.SingleNodeConfiguration, astInstallationPath, astRole string) []string {
	// If log location is empty, set it to defaultLogLocation
	if configuration.Log.Location == "" {
		configuration.Log.Location = defaultLogLocation
	}

	return []string{
		envKeyAndValue(astInstallationPathEnv, astInstallationPath),
		envKeyAndValue(params.AstRoleEnv, astRole),

		envKeyAndValue(dbHostEnv, configuration.Database.Host),
		envKeyAndValue(dbPortEnv, configuration.Database.Port),
		envKeyAndValue(dbUserEnv, configuration.Database.Username),
		envKeyAndValue(dbPasswordEnv, configuration.Database.Password),
		envKeyAndValue(dbInstanceEnv, configuration.Database.Instance),

		envKeyAndValue(traefikPort, configuration.Network.EntrypointPort),
		envKeyAndValue(privateKeyPath, configuration.Network.TLS.PrivateKeyPath),
		envKeyAndValue(certificatePath, configuration.Network.TLS.CertificatePath),
		envKeyAndValue(externalHostname, configuration.Network.ExternalHostname),

		envKeyAndValue(logLevel, configuration.Log.Level),
		envKeyAndValue(logLocation, configuration.Log.Location),
		envKeyAndValue(logRotationCount, configuration.Log.Rotation.Count),
		envKeyAndValue(logRotationMaxSizeMB, configuration.Log.Rotation.MaxSizeMB),
	}
}

func envKeyAndValue(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}
