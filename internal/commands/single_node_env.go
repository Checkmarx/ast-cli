package commands

import (
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/config"
)

const (
	astInstallationPathEnv = "AST_INSTALLATION_PATH"
	executionTypeEnv       = "EXECUTION_TYPE"

	// Configurable database environment variables
	dbHostEnv     = "DATABASE_HOST"
	dbPortEnv     = "DATABASE_PORT"
	dbUserEnv     = "DATABASE_USER"
	dbPasswordEnv = "DATABASE_PASSWORD"
	dbInstanceEnv = "DATABASE_DB"

	// Configurable network environment variables
	traefikPort      = "TRAEFIK_PORT"
	traefikTLSPort   = "TRAEFIK_SSL_PORT"
	privateKeyPath   = "TLS_PRIVATE_KEY_PATH"
	certificatePath  = "TLS_CERTIFICATE_PATH"
	fqdn             = "FQDN"
	externalAccessIP = "EXTERNAL_ACCESS_IP"

	// Configurable object store environment variables
	objectStoreAccessKeyID     = "OBJECT_STORE_ACCESS_KEY_ID"
	objectStoreSecretAccessKey = "OBJECT_STORE_SECRET_ACCESS_KEY"

	// Configurable message queue environment variables
	messageQueueUsername = "NATS_USERNAME"
	messageQueuePassword = "NATS_PASSWORD"

	// Configurable access control environment variables
	accessControlUsername = "KEYCLOAK_USER"
	accessControlPassword = "KEYCLOAK_PASSWORD"

	// Configurable logging environment variables
	logLevel             = "LOG_LEVEL"
	logRotationAgeDays   = "LOG_ROTATION_AGE_DAYS"
	logRotationMaxSizeMB = "LOG_ROTATION_MAX_SIZE_MB"
)

func getEnvVarsForCommand(configuration *config.SingleNodeConfiguration, astInstallationPath string) []string {
	return []string{
		envKeyAndValue(astInstallationPathEnv, astInstallationPath),
		envKeyAndValue(executionTypeEnv, configuration.Execution.Type),

		envKeyAndValue(dbHostEnv, configuration.Database.Host),
		envKeyAndValue(dbPortEnv, configuration.Database.Port),
		envKeyAndValue(dbUserEnv, configuration.Database.Username),
		envKeyAndValue(dbPasswordEnv, configuration.Database.Password),
		envKeyAndValue(dbInstanceEnv, configuration.Database.Name),

		envKeyAndValue(traefikPort, configuration.Network.EntrypointPort),
		envKeyAndValue(traefikTLSPort, configuration.Network.EntrypointTLSPort),
		envKeyAndValue(privateKeyPath, configuration.Network.TLS.PrivateKeyPath),
		envKeyAndValue(certificatePath, configuration.Network.TLS.CertificatePath),
		envKeyAndValue(fqdn, configuration.Network.FullyQualifiedDomainName),
		envKeyAndValue(externalAccessIP, configuration.Network.ExternalAccessIP),

		envKeyAndValue(objectStoreAccessKeyID, configuration.ObjectStore.AccessKeyID),
		envKeyAndValue(objectStoreSecretAccessKey, configuration.ObjectStore.SecretAccessKey),

		envKeyAndValue(messageQueueUsername, configuration.MessageQueue.Username),
		envKeyAndValue(messageQueuePassword, configuration.MessageQueue.Password),

		envKeyAndValue(accessControlUsername, configuration.AccessControl.Username),
		envKeyAndValue(accessControlPassword, configuration.AccessControl.Password),

		envKeyAndValue(logLevel, configuration.Log.Level),
		envKeyAndValue(logRotationAgeDays, configuration.Log.Rotation.MaxAgeDays),
		envKeyAndValue(logRotationMaxSizeMB, configuration.Log.Rotation.MaxSizeMB),
	}
}

func envKeyAndValue(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}
