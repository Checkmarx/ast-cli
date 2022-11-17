package commands

import (
	"fmt"
	"log"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	failedCreatingClient = "failed creating client"
	pleaseProvideFlag    = "%s: Please provide %s flag"
	SuccessAuthValidate  = "Successfully authenticated to AST server!"
	adminClientID        = "ast-app"
	adminClientSecret    = "1d71c35c-818e-4ee8-8fb1-d6cbf8fe2e2a"
	FailedAuthError      = "Failed to authenticate - please provide client-id, client-secret and base-uri or apikey"
)

var (
	RoleSlice = []string{
		"ast-admin",
		"ast-scanner",
	}
	roleSet = map[string]bool{
		"ast-admin":   true,
		"ast-scanner": true,
	}
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
		Example: heredoc.Doc(
			`
			$ cx auth validate
			Successfully authenticated to AST server!
			$ cx auth register -u <Username> -p <Password> --base-uri https://<Keycloak server URI>
			CX_CLIENT_ID=XX
			CX_CLIENT_SECRET=XX
		`,
		),
		Annotations: map[string]string{
			"utils:env": heredoc.Doc(
				`
				See 'cx utils env' for the list of supported environment variables
			`,
			),
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68627-auth.html
			`,
			),
		},
	}
	createClientCmd := &cobra.Command{
		Use:     "register",
		Short:   "Register new OAuth2 client for ast",
		Long:    "Register new OAuth2 client and outputs its generated credentials in the format <key>=<value>",
		Example: "$ cx auth register -u <Username> -p <Password> -r ast-admin,ast-scanner",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68627-auth.html#UUID-c64cdceb-1072-ca20-aa7d-2ba9fd0c4160
			`,
			),
		},
		RunE: runRegister(authWrapper),
	}
	createClientCmd.PersistentFlags().StringP(
		params.UsernameFlag, params.UsernameSh, "", "Username for Ast user that privileges to "+
			"create clients",
	)
	createClientCmd.PersistentFlags().StringP(
		params.PasswordFlag, params.PasswordSh, "", "Password for Ast user that privileges to "+
			"create clients",
	)
	createClientCmd.PersistentFlags().StringP(
		params.ClientDescriptionFlag,
		params.ClientDescriptionSh,
		"",
		"A client description",
	)
	createClientCmd.PersistentFlags().StringSliceP(
		params.ClientRolesFlag, params.ClientRolesSh, []string{},
		fmt.Sprintf("A list of roles of the client %v", RoleSlice),
	)
	markFlagAsRequired(createClientCmd, params.ClientRolesFlag)

	validLoginCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validates authentication",
		Long:  "Validates if CLI is able to communicate with AST",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68627-auth.html#UUID-01803060-8f64-8090-c956-2b505b1d4b61
			`,
			),
		},
		RunE: validLogin(),
	}
	authCmd.AddCommand(createClientCmd, validLoginCmd)
	return authCmd
}

func validLogin() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		clientID := viper.GetString(params.AccessKeyIDConfigKey)
		clientSecret := viper.GetString(params.AccessKeySecretConfigKey)
		apiKey := viper.GetString(params.AstAPIKey)
		if (clientID != "" && clientSecret != "") || apiKey != "" {
			authWrapper := wrappers.NewAuthHTTPWrapper()
			authWrapper.SetPath(viper.GetString(params.ScansPathKey))
			err := authWrapper.ValidateLogin()
			if err != nil {
				return errors.Errorf("%s", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), SuccessAuthValidate)
			return nil
		}
		return errors.Errorf(FailedAuthError)
	}
}

func runRegister(authWrapper wrappers.AuthWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString(params.UsernameFlag)
		if username == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, params.UsernameFlag)
		}
		viper.Set(params.UsernameFlag, username)

		password, _ := cmd.Flags().GetString(params.PasswordFlag)
		if password == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingClient, params.PasswordFlag)
		}
		viper.Set(params.PasswordFlag, password)

		roles, _ := cmd.Flags().GetStringSlice(params.ClientRolesFlag)
		err := validateRoles(roles)
		if err != nil {
			return err
		}

		description, _ := cmd.Flags().GetString(params.ClientDescriptionFlag)
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
			log.Println("Could not create OAuth2 credentials!")
			return err
		}

		if errorMsg != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedCreatingClient, errorMsg.Code, errorMsg.Message)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", params.AccessKeyIDEnv, generatedClientID)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", params.AccessKeySecretEnv, generatedClientSecret)
		return nil
	}
}

func validateRoles(roles []string) error {
	if len(roles) == 0 {
		return errors.Errorf(pleaseProvideFlag, failedCreatingClient, params.ClientRolesFlag)
	}
	for _, role := range roles {
		if !roleSet[role] {
			return errors.Errorf("Invalid role found, please input from %v", RoleSlice)
		}
	}
	return nil
}
