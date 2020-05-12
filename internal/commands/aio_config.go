package commands

import (
	"github.com/checkmarxDev/ast-cli/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	// Configurable database environment variables
	dbHostEnv     = "DATABASE_HOST"
	dbPortEnv     = "DATABASE_PORT"
	dbUserEnv     = "DATABASE_USER"
	dbPasswordEnv = "DATABASE_PASSWORD"
	dbInstanceEnv = "DATABASE_DB"

	// Configurable network environment variables
	traefikPort    = "TRAEFIK_PORT"
	traefikTLSPort = "TRAEFIK_SSL_PORT"

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
	logLevel = "LOG_LEVEL"
)

func mergeConfigurationWithEnv(configuration *config.AIOConfiguration, dotEnvFilePath string) error {
	viperInst := viper.New()
	viperInst.SetConfigFile(dotEnvFilePath)
	viperInst.SetConfigType("env")
	err := viperInst.ReadInConfig() // Find and read the config file
	if err != nil {
		return errors.Wrapf(err, "%s: Failed to open .env file", failedInstallingAIO)
	}
	// Overriding database environment variables
	setIfNotEmpty(viperInst, dbHostEnv, configuration.Database.Host)
	setIfNotEmpty(viperInst, dbPortEnv, configuration.Database.Port)
	setIfNotEmpty(viperInst, dbUserEnv, configuration.Database.Username)
	setIfNotEmpty(viperInst, dbPasswordEnv, configuration.Database.Password)
	setIfNotEmpty(viperInst, dbInstanceEnv, configuration.Database.Name)

	// Overriding network environment variables
	setIfNotEmpty(viperInst, traefikPort, configuration.Network.EntrypointPort)
	setIfNotEmpty(viperInst, traefikTLSPort, configuration.Network.EntrypointTLSPort)

	// Overriding object store environment variables
	setIfNotEmpty(viperInst, objectStoreAccessKeyID, configuration.ObjectStore.AccessKeyID)
	setIfNotEmpty(viperInst, objectStoreSecretAccessKey, configuration.ObjectStore.SecretAccessKey)

	// Overriding message queue environment variables
	setIfNotEmpty(viperInst, messageQueueUsername, configuration.MessageQueue.Username)
	setIfNotEmpty(viperInst, messageQueuePassword, configuration.MessageQueue.Password)

	// Overriding access control environment variables
	setIfNotEmpty(viperInst, accessControlUsername, configuration.AccessControl.Username)
	setIfNotEmpty(viperInst, accessControlPassword, configuration.AccessControl.Password)

	// Overriding logging environment variables
	setIfNotEmpty(viperInst, logLevel, configuration.Log.Level)

	err = viperInst.WriteConfigAs(dotEnvFilePath)
	if err != nil {
		return errors.Wrapf(err, "%s: Failed to write modified .env file", failedInstallingAIO)
	}

	return nil
}

func setIfNotEmpty(viperInst *viper.Viper, key, value string) {
	if value != "" {
		viperInst.Set(key, value)
	}
}
