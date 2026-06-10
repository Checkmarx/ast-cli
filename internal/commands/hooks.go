package commands

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewHooksCommand creates the hooks command with pre-commit subcommand
func NewHooksCommand(jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper, realtimeScannerWrapper wrappers.RealtimeScannerWrapper) *cobra.Command {
	hooksCmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage Git hooks and AI coding agent hooks",
		Long:  "The hooks command manages Git hooks for secret detection and AI coding agent hooks for Claude, Cursor, Windsurf, Factory Droid, and Gemini.",
		Example: heredoc.Doc(
			`
			$ cx hooks pre-commit secrets-install-git-hook
			$ cx hooks pre-commit secrets-scan
			$ cx hooks pre-receive secrets-scan
			$ cx hooks agenthooks install
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
                https://checkmarx.com/resource/documents/en/34965-365503-hooks.html
            `,
			),
		},
	}

	// Add pre-commit, pre-receive, and agenthooks management subcommands
	hooksCmd.AddCommand(PreCommitCommand(jwtWrapper, featureFlagsWrapper))
	hooksCmd.AddCommand(PreReceiveCommand(jwtWrapper, featureFlagsWrapper))
	hooksCmd.AddCommand(NewAgentHooksCommand())

	// Register all hidden hook dispatch subcommands so that cx itself acts as
	// the hook binary. Agents invoke: cx hooks <route-name>
	// e.g. cx hooks claude-pre-tool-use
	for _, dispatchCmd := range HookDispatchCommands(jwtWrapper, featureFlagsWrapper, realtimeScannerWrapper) {
		hooksCmd.AddCommand(dispatchCmd)
	}

	return hooksCmd
}

func validateLicense(jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) error {
	scsLicensingV2Flag, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.ScsLicensingV2Enabled)
	var licenseName string
	if scsLicensingV2Flag.Status {
		licenseName = params.SecretDetectionLabel
	} else {
		licenseName = params.EnterpriseSecretsLabel
	}

	logger.PrintIfVerbose("hooks: checking license for " + licenseName)

	allowed, err := jwtWrapper.IsAllowedEngine(licenseName)
	if err != nil {
		logger.PrintIfVerbose("hooks: authentication failed during license check - " + err.Error())
		return errors.Wrapf(err, "Failed checking license")
	}
	if !allowed {
		logger.PrintIfVerbose("hooks: license validation failed - " + licenseName + " not found in allowed engines")
		return errors.Errorf("Error: License validation failed. Please verify your CxOne license includes %s.", licenseName)
	}
	logger.PrintIfVerbose("hooks: license validated successfully for " + licenseName)
	return nil
}
