package commands

import (
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedCreatingClient = "failed creating client"
	pleaseProvideFlag    = "%s: Please provide %s flag"
	adminClientID        = "ast-app"
	adminClientSecret    = "1d71c35c-818e-4ee8-8fb1-d6cbf8fe2e2a"
)

type ClientCreated struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

func NewAuthCommand(authWrapper wrappers.AuthWrapper) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}
	createClientCmd := &cobra.Command{
		Use:   "register",
		Short: "Register new oath2 client for ast",
		Long: "Register new oath2 client and outputs its credentials in the format <key>=<value>.\n" +
			"If you wish to use this client with the cli set those credentials as environment variables.\n" +
			"On Linux just wrap this command with eval e.g: \"eval $(ast auth register -u <username> -p <password>)\".\n" +
			"On Windows use the SET command with the outputs credentials",
		RunE: runRegister(authWrapper),
	}
	createClientCmd.PersistentFlags().StringP(clientDescriptionFlag, clientDescriptionSh, "", "A client description")
	createClientCmd.PersistentFlags().StringP(usernameFlag, usernameSh, "", "Username for a user that privilege to "+
		"create client")
	createClientCmd.PersistentFlags().StringP(passwordFlag, passwordSh, "", "Password for the user")
	createClientCmd.PersistentFlags().StringSliceP(clientRolesFlag, clientRolesSh, []string{"ast-admin"},
		"A list of roles of the client")
	authCmd.AddCommand(createClientCmd)
	return authCmd
}

func runRegister(authWrapper wrappers.AuthWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString(usernameFlag)
		if username == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, usernameFlag)
		}

		password, _ := cmd.Flags().GetString(passwordFlag)
		if password == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, passwordFlag)
		}

		roles, _ := cmd.Flags().GetStringSlice(clientRolesFlag)
		description, _ := cmd.Flags().GetString(clientDescriptionFlag)
		generatedClientID := "ast-plugins-" + uuid.New().String()
		generatedClientSecret := uuid.New().String()
		client := &wrappers.Oath2Client{
			Name:        generatedClientID,
			Roles:       roles,
			Description: description,
			Secret:      generatedClientSecret,
		}
		errorMsg, err := authWrapper.CreateOauth2Client(client, username, password, adminClientID, adminClientSecret)
		if err != nil {
			return errors.Wrapf(err, "%s", failedCreatingClient)
		}

		if errorMsg != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedCreatingClient, errorMsg.Code, errorMsg.Message)
		}

		fmt.Printf("%s=%s\n", params.AccessKeyIDEnv, generatedClientID)
		fmt.Printf("%s=%s\n", params.AccessKeySecretEnv, generatedClientSecret)
		return nil
	}
}
