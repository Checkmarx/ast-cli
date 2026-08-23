package configuration

import (
	"bufio"
	"context"
	stderrors "errors"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"

	"github.com/checkmarx/ast-cli/internal/configfile"
	"github.com/checkmarx/ast-cli/internal/credentialstore"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const configDirName = "/.checkmarx"
const obfuscateLimit = 4
const homeDirectoryPermissions = 0700

func PromptConfiguration() {
	reader := bufio.NewReader(os.Stdin)
	baseURI := viper.GetString(params.BaseURIKey)
	baseURISrc := viper.GetString(params.BaseURIKey)
	baseAuthURI := viper.GetString(params.BaseAuthURIKey)
	accessKeySecret := resolveSecretForPrompt(params.AccessKeySecretConfigKey)
	accessKey := viper.GetString(params.AccessKeyIDConfigKey)
	accessAPIKey := resolveSecretForPrompt(params.AstAPIKey)
	tenant := viper.GetString(params.TenantKey)
	fmt.Print("Setup guide: https://checkmarx.com/resource/documents/en/34965-68621-checkmarx-one-cli-quick-start-guide.html\n\n")
	// Prompt for Base URI
	fmt.Printf("AST Base URI [%s]: ", baseURI)
	baseURI, _ = reader.ReadString('\n')
	baseURI = strings.Replace(baseURI, "\n", "", -1)
	baseURI = strings.Replace(baseURI, "\r", "", -1)
	if len(baseURI) > 0 {
		setConfigPropertyQuiet(params.BaseURIKey, baseURI)
	}
	// Prompt for Base Auth URI
	if len(baseAuthURI) < 1 {
		baseAuthURI = baseURISrc
	}
	fmt.Printf("AST Base Auth URI (IAM) [%s]: ", baseAuthURI)
	baseAuthURI, _ = reader.ReadString('\n')
	baseAuthURI = strings.Replace(baseAuthURI, "\n", "", -1)
	baseAuthURI = strings.Replace(baseAuthURI, "\r", "", -1)
	if len(baseAuthURI) > 0 {
		setConfigPropertyQuiet(params.BaseAuthURIKey, baseAuthURI)
	}
	// Prompt for tenant name
	fmt.Printf("AST Tenant [%s]: ", tenant)
	tenant, _ = reader.ReadString('\n')
	tenant = strings.Replace(tenant, "\n", "", -1)
	tenant = strings.Replace(tenant, "\r", "", -1)
	if len(tenant) > 0 {
		setConfigPropertyQuiet(params.TenantKey, tenant)
	}
	// Prompt for access credentials type
	fmt.Printf("Do you want to use API Key authentication? (Y/N): ")
	authType, _ := reader.ReadString('\n')
	authType = strings.Replace(authType, "\n", "", -1)
	authType = strings.Replace(authType, "\r", "", -1)
	if strings.EqualFold(authType, "Y") {
		fmt.Printf("AST API Key [%s]: ", ObfuscateString(accessAPIKey))
		accessAPIKey, _ = reader.ReadString('\n')
		accessAPIKey = strings.Replace(accessAPIKey, "\n", "", -1)
		accessAPIKey = strings.Replace(accessAPIKey, "\r", "", -1)
		if len(accessAPIKey) > 0 {
			storeProperty(params.AstAPIKey, accessAPIKey)
			setConfigPropertyQuiet(params.AccessKeyIDConfigKey, "")
			storeProperty(params.AccessKeySecretConfigKey, "")
		}
	} else {
		fmt.Printf("Checkmarx One Client ID [%s]: ", ObfuscateString(accessKey))
		accessKey, _ = reader.ReadString('\n')
		accessKey = strings.Replace(accessKey, "\n", "", -1)
		accessKey = strings.Replace(accessKey, "\r", "", -1)
		if len(accessKey) > 0 {
			setConfigPropertyQuiet(params.AccessKeyIDConfigKey, accessKey)
			storeProperty(params.AstAPIKey, "")
		}
		fmt.Printf("Client Secret [%s]: ", ObfuscateString(accessKeySecret))
		accessKeySecret, _ = reader.ReadString('\n')
		accessKeySecret = strings.Replace(accessKeySecret, "\n", "", -1)
		accessKeySecret = strings.Replace(accessKeySecret, "\r", "", -1)
		if len(accessKeySecret) > 0 {
			storeProperty(params.AccessKeySecretConfigKey, accessKeySecret)
			storeProperty(params.AstAPIKey, "")
		}
	}
}

// PromptAuthConnection prompts for base-uri, base-auth-uri, and tenant (what cx auth
// login needs), like cx configure. Blank keeps the default; non-interactive stdin sets nothing.
func PromptAuthConnection() {
	reader := bufio.NewReader(os.Stdin)
	baseURI := viper.GetString(params.BaseURIKey)
	baseURISrc := baseURI
	baseAuthURI := viper.GetString(params.BaseAuthURIKey)
	tenant := viper.GetString(params.TenantKey)

	fmt.Printf("AST Base URI [%s]: ", baseURI)
	if v := readLine(reader); v != "" {
		setConfigPropertyQuiet(params.BaseURIKey, v)
	}
	if baseAuthURI == "" {
		baseAuthURI = baseURISrc
	}
	fmt.Printf("AST Base Auth URI (IAM) [%s]: ", baseAuthURI)
	if v := readLine(reader); v != "" {
		setConfigPropertyQuiet(params.BaseAuthURIKey, v)
	}
	fmt.Printf("AST Tenant [%s]: ", tenant)
	if v := readLine(reader); v != "" {
		setConfigPropertyQuiet(params.TenantKey, v)
	}
}

func readLine(reader *bufio.Reader) string {
	s, _ := reader.ReadString('\n')
	return strings.TrimSpace(s)
}

// ObfuscateString masks all but the last four characters of a secret value.
func ObfuscateString(str string) string {
	if len(str) > obfuscateLimit {
		return "******" + str[len(str)-4:]
	} else if len(str) > 1 {
		return "******"
	} else {
		return ""
	}
}

func setConfigPropertyQuiet(propName, propValue string) {
	viper.Set(propName, propValue)
	// You should be able to  call WriteConfig() but it will fail if the
	// config file doesn't already exist, this is a known viper bug.
	// SafeWriteConfig() will not update files but it will create them, combined
	// this code will successfully update files.
	if viperErr := viper.SafeWriteConfig(); viperErr != nil {
		err := viper.WriteConfig()
		if err != nil {
			fmt.Println("Error writing config file", err)
		}
	}
}

func SetConfigProperty(propName, propValue string) error {
	displayValue := propValue
	if credentialstore.IsSecret(strings.ToLower(propName)) {
		displayValue = ObfuscateString(propValue)
	}
	fmt.Println("Setting property [", propName, "] to value [", displayValue, "]")
	return setConfigProperty(strings.ToLower(propName), propValue)
}

func storeProperty(propName, propValue string) {
	if err := setConfigProperty(propName, propValue); err != nil {
		fmt.Println("Error storing property", propName, err)
	}
}

func setConfigProperty(propName, propValue string) error {
	credentialName, ok := credentialstore.CredentialForViperKey(propName)
	if !ok {
		setConfigPropertyQuiet(propName, propValue)
		return nil
	}
	resolver := credentialstore.Default()
	var err error
	if propValue == "" {
		err = resolver.Clear(context.Background(), credentialName)
		if stderrors.Is(err, credentialstore.ErrNotFound) {
			err = nil
		}
	} else {
		err = resolver.Store(context.Background(), credentialName, propValue)
	}
	if err != nil {
		return fmt.Errorf("storing %s: %w", propName, err)
	}
	if !resolver.StoresInConfigFile() {
		removePlaintextEntryQuietly(propName)
	}
	return nil
}

func removePlaintextEntryQuietly(propName string) {
	credentialName, ok := credentialstore.CredentialForViperKey(propName)
	if !ok {
		return
	}
	if err := credentialstore.Default().RemoveConfigFileEntry(credentialName); err != nil {
		logger.PrintfIfVerbose("could not remove %s from config file: %v", propName, err)
	}
}

func resolveSecretForPrompt(viperKey string) string {
	credentialName, ok := credentialstore.CredentialForViperKey(viperKey)
	if !ok {
		return ""
	}
	value, err := credentialstore.Resolve(credentialName)
	if err != nil {
		return ""
	}
	return value
}

func LoadConfiguration() error {
	configFilePath := viper.GetString(params.ConfigFilePathKey)
	if configFilePath == "" {
		// Read directly so consumers without viper bindings (tests, embedded
		// runs) still honor the environment variable.
		configFilePath = os.Getenv(params.ConfigFilePathEnv)
	}

	if configFilePath != "" {
		err := validateConfigFile(configFilePath)
		if err != nil {
			return err
		}
		viper.SetConfigFile(configFilePath)
		if err = viper.ReadInConfig(); err != nil {
			return errors.New("An error occurred while accessing the file or environment variable. Please verify the CLI configuration file")
		}
	} else {
		usr, err := user.Current()
		if err != nil {
			log.Fatal("Cannot file home directory.", err)
		}
		fullPath := usr.HomeDir + configDirName
		verifyConfigDir(fullPath)
		viper.SetConfigFile(fullPath + "/checkmarxcli.yaml")
		_ = viper.ReadInConfig()
	}
	return nil
}

func validateConfigFile(configFilePath string) error {
	info, err := os.Stat(configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("The specified file does not exist. Please check the path and ensure the CLI configuration file is available.")
		}
		return fmt.Errorf("An error occurred while accessing the file or environment variable. Please verify the CLI configuration file")
	}

	if info.IsDir() {
		return fmt.Errorf("The specified path points to a directory, not a file. Please provide a valid CLI configuration file path.")
	}

	file, err := os.OpenFile(configFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("Access to the specified file is restricted. Please ensure you have the necessary permissions to access the CLI configuration file")
	}
	defer file.Close()

	return nil
}

func SafeWriteSingleConfigKey(configFilePath, key string, value int) error {
	return configfile.SetKey(configFilePath, key, value)
}

func SafeWriteSingleConfigKeyString(configFilePath, key string, value string) error {
	return configfile.SetKey(configFilePath, key, value)
}

// LoadConfig loads the configuration from a file. If the file does not exist
// or is empty, it returns an empty map.
func LoadConfig(path string) (map[string]interface{}, error) {
	return configfile.Load(path)
}

// SaveConfig writes the configuration to a file.
func SaveConfig(path string, config map[string]interface{}) error {
	return configfile.Save(path, config)
}

func GetConfigFilePath() (string, error) {
	configFilePath := viper.GetString(params.ConfigFilePathKey)
	if configFilePath == "" {
		// Test-isolation seam: harnesses set the env var without viper bindings.
		configFilePath = os.Getenv(params.ConfigFilePathEnv)
	}

	if configFilePath == "" {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("error getting current user: %w", err)
		}
		configFilePath = usr.HomeDir + configDirName + "/checkmarxcli.yaml"
	}
	return configFilePath, nil
}

func verifyConfigDir(fullPath string) {
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		logger.PrintfIfVerbose("Creating configuration file at default directory path")
		err = os.Mkdir(fullPath, homeDirectoryPermissions)
		if err != nil {
			log.Fatal("Cannot file home directory.", err)
		}
	}
}

func ShowConfiguration() {
	fmt.Println("Current Effective Configuration")

	fmt.Printf("%30v", "BaseURI: ")
	fmt.Println(viper.GetString(params.BaseURIKey))
	fmt.Printf("%30v", "BaseAuthURIKey: ")
	fmt.Println(viper.GetString(params.BaseAuthURIKey))
	fmt.Printf("%30v", "Checkmarx One Tenant: ")
	fmt.Println(viper.GetString(params.TenantKey))
	fmt.Printf("%30v", "Client ID: ")
	fmt.Println(viper.GetString(params.AccessKeyIDConfigKey))
	fmt.Printf("%30v", "Client Secret: ")
	fmt.Println(ObfuscateString(resolveSecretForPrompt(params.AccessKeySecretConfigKey)))
	fmt.Printf("%30v", "APIKey: ")
	fmt.Println(ObfuscateString(resolveSecretForPrompt(params.AstAPIKey)))
	fmt.Printf("%30v", "Proxy: ")
	fmt.Println(viper.GetString(params.ProxyKey))
}
