package wrappers

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/spf13/viper"
)

const defaultProfileName = "default"
const obfuscateLimit = 4

func PromptConfiguration() {
	reader := bufio.NewReader(os.Stdin)
	baseURI := viper.GetString(params.BaseURIKey)
	accessKeySecret := viper.GetString(params.AccessKeySecretConfigKey)
	accessKey := viper.GetString(params.AccessKeyIDConfigKey)
	proxyKey := viper.GetString(params.ProxyKey)
	fmt.Printf("AST Base URI [%s]: ", baseURI)
	baseURI, _ = reader.ReadString('\n')
	baseURI = strings.Replace(baseURI, "\r\n", "", -1)
	baseURI = strings.Replace(baseURI, "\n", "", -1)
	if len(baseURI) > 0 {
		setConfigPropertyQuiet(params.BaseURIKey, baseURI)
	}
	fmt.Printf("AST Access Key [%s]: ", obfuscateString(accessKey))
	accessKey, _ = reader.ReadString('\n')
	accessKey = strings.Replace(accessKey, "\r\n", "", -1)
	accessKey = strings.Replace(accessKey, "\n", "", -1)
	if len(accessKey) > 0 {
		setConfigPropertyQuiet(params.AccessKeyIDConfigKey, accessKey)
	}
	fmt.Printf("AST Key Secret [%s]: ", obfuscateString(accessKeySecret))
	accessKeySecret, _ = reader.ReadString('\n')
	accessKeySecret = strings.Replace(accessKeySecret, "\r\n", "", -1)
	accessKeySecret = strings.Replace(accessKeySecret, "\n", "", -1)
	if len(accessKeySecret) > 0 {
		setConfigPropertyQuiet(params.AccessKeySecretConfigKey, accessKeySecret)
	}
	fmt.Printf("Proxy server [%s]: ", proxyKey)
	proxyKey, _ = reader.ReadString('\n')
	proxyKey = strings.Replace(proxyKey, "\r\n", "", -1)
	proxyKey = strings.Replace(proxyKey, "\n", "", -1)
	if len(proxyKey) > 0 {
		setConfigPropertyQuiet(params.ProxyKey, proxyKey)
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
	profile := findProfile()
	viper.AddConfigPath("$HOME/.checkmarx")
	configFile := ".checkmarxcli"
	if profile != defaultProfileName {
		configFile += "_"
		configFile += profile
	}
	viper.SetConfigName(configFile)
	viper.SetConfigType("yaml")
	_ = viper.ReadInConfig()
}

func findProfile() string {
	profileName := defaultProfileName
	for idx, b := range os.Args {
		if b == "--profile" {
			profileIdx := idx + 1
			if len(os.Args) > profileIdx {
				profileName = os.Args[profileIdx]
				fmt.Println("Using custom profile: ", profileName)
			}
		}
	}
	return profileName
}

func ShowConfiguration() {
	fmt.Println("Current Effective Configuration")
	baseURI := viper.GetString(params.BaseURIKey)
	fmt.Println("\tBaseURI: ", baseURI)
}
