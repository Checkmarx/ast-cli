package wrappers

import (
	"fmt"
	"os"

	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/spf13/viper"
)

const defaultProfileName = "default"

func SetConfigProperty(propName, propValue string) {
	fmt.Println("Setting property [", propName, "] to value [", propValue, "]")
	viper.Set(propName, propValue)
	// You should be able to  call WriteConfig() but it will fail if the
	// config file doesn't already exist, this is a known viper bug.
	// SafeWriteConfig() will not update files but it will create them, combined
	// this code will successfully update files.
	if viperErr := viper.SafeWriteConfig(); viperErr != nil {
		_ = viper.WriteConfig()
	}
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
	err := viper.ReadInConfig()
	if err != nil {
		//fmt.Println(err)
	}
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
