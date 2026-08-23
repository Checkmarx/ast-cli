package util

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/credentialstore"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const formatString = "%30v: %s\n"

// proxyEnvName is used because the proxy is bound outside EnvVarsBinds.
const proxyEnvName = params.ProxyEnv

// NewEnvCheckCommand returns the `cx utils env` command.
func NewEnvCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Shows the current profiles configuration properties",
		Example: heredoc.Doc(
			`
			$ cx utils env
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68653-utils.html#UUID-f7245425-72b9-9854-a60a-a9f37e0173d9
			`,
			),
		},
		RunE: runEnvChecks(),
	}
	return cmd
}

type envBinding struct {
	key string
	env string
}

// configurableEnvBindings maps each property in Properties to its uppercase
// environment variable name, in declaration order. Secret properties are
// appended explicitly: they are no longer bound to environment variables but
// remain listed (obfuscated) configuration entries.
func configurableEnvBindings() []envBinding {
	bindings := make([]envBinding, 0, len(Properties)+3)
	for _, bind := range params.EnvVarsBinds {
		if Properties[bind.Key] {
			bindings = append(bindings, envBinding{key: bind.Key, env: bind.Env})
		}
	}
	for _, bind := range []envBinding{
		{key: params.AstAPIKey, env: params.AstAPIKeyEnv},
		{key: params.AccessKeySecretConfigKey, env: params.AccessKeySecretEnv},
	} {
		if Properties[bind.key] {
			bindings = append(bindings, bind)
		}
	}
	if Properties[params.ProxyKey] {
		bindings = append(bindings, envBinding{key: params.ProxyKey, env: proxyEnvName})
	}
	return bindings
}

func runEnvChecks() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("\nDetected Environment Variables:\n\n")
		for _, bind := range configurableEnvBindings() {
			value := effectivePropertyValue(bind.key)
			if _, isSecret := credentialstore.CredentialForViperKey(bind.key); isSecret {
				value = configuration.ObfuscateString(value)
			}
			fmt.Printf(formatString, bind.env, value)
		}
		return nil
	}
}

// effectivePropertyValue returns the current value of a configuration
// property: secrets resolve through the credential store, the rest via viper.
func effectivePropertyValue(viperKey string) string {
	credentialName, ok := credentialstore.CredentialForViperKey(viperKey)
	if !ok {
		return viper.GetString(viperKey)
	}
	value, err := credentialstore.Resolve(credentialName)
	if err != nil {
		return ""
	}
	return value
}
