package credentialstore

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

// LoadStoredSecrets copies stored secrets into viper so every viper.GetString read
// path keeps working. Env wins over the store; explicit flags are re-asserted later
// by CheckPreferredCredentials.
func LoadStoredSecrets() {
	loadStoredSecretIfEnvAbsent(params.AstAPIKey, params.AstAPIKeyEnv)
	loadStoredSecretIfEnvAbsent(params.AccessKeySecretConfigKey, params.AccessKeySecretEnv)
}

func loadStoredSecretIfEnvAbsent(key, envName string) {
	if os.Getenv(envName) != "" {
		return
	}
	if v, err := Default.GetSecret(key); err == nil && v != "" {
		viper.Set(key, v)
	}
}
