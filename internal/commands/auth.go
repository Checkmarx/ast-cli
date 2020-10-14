package commands

import (
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const failedCreatingClient = "failed creating client"
const pleaseProvideFlag = "%s: Please provide %s flag"

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
		Use:   "create-oath2-client <client id>",
		Short: "Create oath2 client for ast",
		RunE:  runCreateOath2ClientCommand(authWrapper),
	}
	createClientCmd.PersistentFlags().StringP(clientDescriptionFlag, clientDescriptionSh, "", "A client description")
	createClientCmd.PersistentFlags().StringSliceP(clientRolesFlag, clientRolesSh, []string{}, "A list of roles of the client")
	createClientCmd.PersistentFlags().StringP(usernameFlag, usernameSh, "", "Username for a user that privilege to "+
		"create client")
	createClientCmd.PersistentFlags().StringP(passwordFlag, passwordSh, "", "Password for the user")
	createClientCmd.PersistentFlags().StringP(adminClientIDFlag, adminClientIDSh, "", "Admin client id")
	createClientCmd.PersistentFlags().StringP(adminClientSecretFlag, adminClientSecretSh, "", "Admin client secret")
	authCmd.AddCommand(createClientCmd)
	return authCmd
}

func runCreateOath2ClientCommand(authWrapper wrappers.AuthWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a client id", failedCreatingClient)
		}

		clientID := args[0]
		username, _ := cmd.Flags().GetString(usernameFlag)
		if username == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, usernameFlag)
		}

		password, _ := cmd.Flags().GetString(passwordFlag)
		if password == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, passwordFlag)
		}

		adminClientID, _ := cmd.Flags().GetString(adminClientIDFlag)
		if adminClientID == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, adminClientIDFlag)
		}

		adminClientSecret, _ := cmd.Flags().GetString(adminClientSecretFlag)
		if adminClientSecret == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, adminClientSecretFlag)
		}

		roles, _ := cmd.Flags().GetStringSlice(clientRolesFlag)
		description, _ := cmd.Flags().GetString(clientDescriptionFlag)
		newClientSecret := uuid.New().String()
		client := &wrappers.Oath2Client{
			Name:        clientID,
			Roles:       roles,
			Description: description,
			Secret:      newClientSecret,
		}
		errorMsg, err := authWrapper.CreateOauth2Client(client, username, password, adminClientID, adminClientSecret)
		if err != nil {
			return errors.Wrapf(err, "%s", failedCreatingClient)
		}

		if errorMsg != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedCreatingClient, errorMsg.Code, errorMsg.Message)
		}

		return Print(cmd.OutOrStdout(), &ClientCreated{ID: clientID, Secret: newClientSecret})
	}
}
