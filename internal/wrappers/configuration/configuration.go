package configuration

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const configDirName = "/.checkmarx"
const obfuscateLimit = 4
const homeDirectoryPermissions = 0700

func PromptConfiguration() {
	reader := bufio.NewReader(os.Stdin)
	baseURI := viper.GetString(params.BaseURIKey)
	baseURISrc := viper.GetString(params.BaseURIKey)
	baseAuthURI := viper.GetString(params.BaseAuthURIKey)
	accessKeySecret := viper.GetString(params.AccessKeySecretConfigKey)
	accessKey := viper.GetString(params.AccessKeyIDConfigKey)
	accessAPIKey := viper.GetString(params.AstAPIKey)
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
		fmt.Printf("AST API Key [%s]: ", obfuscateString(accessAPIKey))
		accessAPIKey, _ = reader.ReadString('\n')
		accessAPIKey = strings.Replace(accessAPIKey, "\n", "", -1)
		accessAPIKey = strings.Replace(accessAPIKey, "\r", "", -1)
		if len(accessAPIKey) > 0 {
			setConfigPropertyQuiet(params.AstAPIKey, accessAPIKey)
			setConfigPropertyQuiet(params.AccessKeyIDConfigKey, "")
			setConfigPropertyQuiet(params.AccessKeySecretConfigKey, "")
		}
	} else {
		fmt.Printf("Checkmarx One Client ID [%s]: ", obfuscateString(accessKey))
		accessKey, _ = reader.ReadString('\n')
		accessKey = strings.Replace(accessKey, "\n", "", -1)
		accessKey = strings.Replace(accessKey, "\r", "", -1)
		if len(accessKey) > 0 {
			setConfigPropertyQuiet(params.AccessKeyIDConfigKey, accessKey)
			setConfigPropertyQuiet(params.AstAPIKey, "")
		}
		fmt.Printf("Client Secret [%s]: ", obfuscateString(accessKeySecret))
		accessKeySecret, _ = reader.ReadString('\n')
		accessKeySecret = strings.Replace(accessKeySecret, "\n", "", -1)
		accessKeySecret = strings.Replace(accessKeySecret, "\r", "", -1)
		if len(accessKeySecret) > 0 {
			setConfigPropertyQuiet(params.AccessKeySecretConfigKey, accessKeySecret)
			setConfigPropertyQuiet(params.AstAPIKey, "")
		}
	}
}

func obfuscateString(str string) string {
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
		_ = viper.WriteConfig()
	}
}

func SetConfigProperty(propName, propValue string) {
	fmt.Println("Setting property [", propName, "] to value [", propValue, "]")
	setConfigPropertyQuiet(propName, propValue)
}

func LoadConfiguration() {
	configFilePath := viper.GetString(params.ConfigFilePathKey)

	if configFilePath != "" {
		err := validateConfigFile(configFilePath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		viper.SetConfigFile(configFilePath)
		if err = viper.ReadInConfig(); err != nil {
			fmt.Println("An error occurred while accessing the file or environment variable. Please verify the CLI configuration file")
			os.Exit(1)
		}
	} else {
		usr, err := user.Current()
		if err != nil {
			log.Fatal("Cannot file home directory.", err)
		}
		fullPath := usr.HomeDir + configDirName
		verifyConfigDir(fullPath)
		viper.AddConfigPath(fullPath)
		configFile := "checkmarxcli"
		viper.SetConfigName(configFile)
		viper.SetConfigType("yaml")
		_ = viper.ReadInConfig()
	}
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
	// Create a file lock
	lock := flock.New(configFilePath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		return errors.Errorf("error acquiring lock: %s", err.Error())
	}
	if !locked {
		return errors.Errorf("could not acquire lock")
	}
	defer func() {
		_ = lock.Unlock()
	}()

	// Load existing configuration or initialize a new one
	config, err := LoadConfig(configFilePath)
	if err != nil {
		return errors.Errorf("error loading config: %s", err.Error())
	}

	// Update the configuration key
	config[key] = value

	// Save the updated configuration back to the file
	if err = SaveConfig(configFilePath, config); err != nil {
		return errors.Errorf("error saving config: %s", err.Error())
	}
	return nil
}

func SafeWriteSingleConfigKeyString(configFilePath, key string, value string) error {
	// Create a file lock
	lock := flock.New(configFilePath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		return errors.Errorf("error acquiring lock: %s", err.Error())
	}
	if !locked {
		return errors.Errorf("could not acquire lock")
	}
	defer func() {
		_ = lock.Unlock()
	}()

	// Load existing configuration or initialize a new one
	config, err := LoadConfig(configFilePath)
	if err != nil {
		return errors.Errorf("error loading config: %s", err.Error())
	}

	// Update the configuration key
	config[key] = value

	// Save the updated configuration back to the file
	if err = SaveConfig(configFilePath, config); err != nil {
		return errors.Errorf("error saving config: %s", err.Error())
	}
	return nil
}

// LoadConfig loads the configuration from a file. If the file does not exist, it returns an empty map.
func LoadConfig(path string) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Return an empty config if the file doesn't exist
		}
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	decoder := yaml.NewDecoder(file)
	if err = decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding YAML: %w", err)
	}
	return config, nil
}

// SaveConfig writes the configuration to a file.
func SaveConfig(path string, config map[string]interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	encoder := yaml.NewEncoder(file)
	if err = encoder.Encode(config); err != nil {
		return fmt.Errorf("error encoding YAML: %w", err)
	}
	return nil
}

func GetConfigFilePath() (string, error) {
	configFilePath := viper.GetString(params.ConfigFilePathKey)

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
		fmt.Println("Creating directory")
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
	fmt.Println(obfuscateString(viper.GetString(params.AccessKeySecretConfigKey)))
	fmt.Printf("%30v", "APIKey: ")
	fmt.Println(obfuscateString(viper.GetString(params.AstAPIKey)))
	fmt.Printf("%30v", "Proxy: ")
	fmt.Println(viper.GetString(params.ProxyKey))
}
