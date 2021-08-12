package commands

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"

	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	failedAuthValidate   = "Failed authentication!"
	failedCreatingClient = "failed creating client"
	pleaseProvideFlag    = "%s: Please provide %s flag"
	SuccessAuthValidate  = "Successfully authenticated to AST server!"
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
		Short: "Validate authentication and create OAuth2 credentials",
		Long:  "Validate authentication and create OAuth2 credentials",
		Example: heredoc.Doc(`
			$ cx auth validate
			Successfully authenticated to AST server!
			$ cx auth register -u <Username> -p <Password> --base-uri https://<Keycloak server URI>
			CX_CLIENT_ID=XX
			CX_CLIENT_SECRET=XX
		`),
		Annotations: map[string]string{
			"utils:env": heredoc.Doc(`
				See 'cx utils env' for the list of supported environment variables
			`),
			"command:doc": heredoc.Doc(`
				https://checkmarx.atlassian.net/wiki/x/LwF4tQ
			`),
		},
	}
	createClientCmd := &cobra.Command{
		Use:     "register",
		Short:   "Register new OAuth2 client for ast",
		Long:    "Register new OAuth2 client and outputs its generated credentials in the format <key>=<value>",
		Example: "$ cx auth register -u <Username> -p <Password>",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(`
				https://checkmarx.atlassian.net/wiki/x/LwF4tQ
			`),
		},
		RunE: runRegister(authWrapper),
	}
	createClientCmd.PersistentFlags().StringP(UsernameFlag, UsernameSh, "", "Username for Ast user that privileges to "+
		"create clients")
	createClientCmd.PersistentFlags().StringP(PasswordFlag, PasswordSh, "", "Password for Ast user that privileges to "+
		"create clients")
	createClientCmd.PersistentFlags().StringP(ClientDescriptionFlag, ClientDescriptionSh, "", "A client description")
	createClientCmd.PersistentFlags().StringSliceP(ClientRolesFlag, ClientRolesSh, []string{"ast-admin"},
		"A list of roles of the client")

	validLoginCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validates authentication",
		Long:  "Validates if CLI is able to communicate with AST",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(`
				https://checkmarx.atlassian.net/wiki/x/LwF4tQ
			`),
		},
		RunE: validLogin(),
	}
	authCmd.AddCommand(createClientCmd, validLoginCmd)
	return authCmd
}

func validLogin() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scansWrapper := wrappers.NewHTTPScansWrapper(viper.GetString(params.ScansPathKey))
		paramsList := make(map[string]string)
		_, _, err := scansWrapper.Get(paramsList)
		if err != nil {
			return errors.Errorf("%s", failedAuthValidate)
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), SuccessAuthValidate)

		return nil
	}
}

func runRegister(authWrapper wrappers.AuthWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString(UsernameFlag)
		if username == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, UsernameFlag)
		}

		password, _ := cmd.Flags().GetString(PasswordFlag)
		if password == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, PasswordFlag)
		}

		roles, _ := cmd.Flags().GetStringSlice(ClientRolesFlag)
		description, _ := cmd.Flags().GetString(ClientDescriptionFlag)
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
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Could not create OAuth2 credentials!")
			return nil
		}

		if errorMsg != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedCreatingClient, errorMsg.Code, errorMsg.Message)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", params.AccessKeyIDEnv, generatedClientID)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", params.AccessKeySecretEnv, generatedClientSecret)
		return nil
	}
}
