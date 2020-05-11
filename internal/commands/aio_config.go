package commands

import (
	"fmt"
	"github.com/checkmarxDev/ast-cli/internal/config"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func mergeConfigurationWithEnv(configuration *config.AIOConfiguration) error {
	executablePath, err := os.Executable()
	if err != nil {
		return errors.Wrapf(err, "failed to merge configuration file with env file")
	}
	executableDir := filepath.Dir(executablePath)
	dotEnvFilePath := path.Join(executableDir, ".env")
	fmt.Println("DOT ENV FILE PATH IS:", dotEnvFilePath)
	dotEnvInput, err := ioutil.ReadFile(dotEnvFilePath)
	if err != nil {
		fmt.Println("failed dot enving")

		return errors.Wrapf(err, "%s: Failed to open .env file", failedInstallingAIO)
	}
	fmt.Println("DOT ENV content is:", string(dotEnvInput))
	return nil
}
