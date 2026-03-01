package commands

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	whoamiTenantClaimKey       = "tenant_name"
	whoamiUserClaimKey         = "name"
	whoamiUserFallbackClaimKey = "sub"
)

type whoamiView struct {
	User   string `json:"user" format:"name:User"`
	Tenant string `json:"tenant" format:"name:Tenant"`
}

func NewWhoamiCommand() *cobra.Command {
	whoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show authenticated user and tenant information",
		Long:  "The whoami command shows authenticated user and tenant information from the current access token",
		Example: heredoc.Doc(
			`
			$ cx whoami
		`,
		),
		RunE: runWhoami,
	}

	addFormatFlag(whoamiCmd, printer.FormatList, printer.FormatJSON, printer.FormatTable)
	return whoamiCmd
}

func runWhoami(cmd *cobra.Command, _ []string) error {
	accessToken, err := wrappers.GetAccessToken()
	if err != nil {
		return err
	}

	user, err := extractWhoamiUser(accessToken)
	if err != nil {
		return errors.Wrap(err, "failed to extract authenticated user from token")
	}

	tenant, err := wrappers.ExtractFromTokenClaims(accessToken, whoamiTenantClaimKey)
	if err != nil {
		return errors.Wrap(err, "failed to extract tenant from token")
	}

	return printByFormat(cmd, whoamiView{
		User:   user,
		Tenant: tenant,
	})
}

func extractWhoamiUser(accessToken string) (string, error) {
	claimKeys := []string{whoamiUserClaimKey, whoamiUserFallbackClaimKey}
	var err error
	for _, claimKey := range claimKeys {
		var user string
		user, err = wrappers.ExtractFromTokenClaims(accessToken, claimKey)
		if err == nil {
			return user, nil
		}
	}
	return "", err
}
