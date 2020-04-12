package commands

import (
	"fmt"
	"strings"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	verboseFlag                   = "verbose"
	verboseFlagSh                 = "v"
	verboseUsage                  = "Verbose mode"
	sourcesFlag                   = "sources"
	sourcesFlagSh                 = "s"
	inputFlag                     = "input"
	inputFlagSh                   = "i"
	inputFileFlag                 = "input-file"
	inputFileFlagSh               = "f"
	limitFlag                     = "limit"
	limitFlagSh                   = "l"
	limitUsage                    = "The number of items to return"
	offsetFlag                    = "offset"
	offsetFlagSh                  = "o"
	offsetUsage                   = "The number of items to skip before collecting the results"
	AccessKeyIDEnv                = "AST_ACCESS_KEY_ID"
	accessKeyIDFlag               = "key"
	accessKeyIDFlagUsage          = "The access key ID for AST"
	AccessKeySecretEnv            = "AST_ACCESS_KEY_SECRET"
	accessKeySecretFlag           = "secret"
	accessKeySecretFlagUsage      = "The access key secret for AST"
	AstAuthenticationURIEnv       = "AST_AUTHENTICATION_URI"
	astAuthenticationURIFlag      = "auth-uri"
	astAuthenticationURIFlagUsage = "The authentication URI for AST"
)

var (
	AccessKeyIDConfigKey          = strings.ToLower(AccessKeyIDEnv)
	AccessKeySecretConfigKey      = strings.ToLower(AccessKeySecretEnv)
	AstAuthenticationURIConfigKey = strings.ToLower(AstAuthenticationURIEnv)
)

// Return an AST CLI root command to execute
func NewAstCLI(scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	resultsWrapper wrappers.ResultsWrapper) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ast",
		Short: "A CLI wrapping Checkmarx AST APIs",
	}

	rootCmd.PersistentFlags().BoolP(verboseFlag, verboseFlagSh, false, verboseUsage)
	rootCmd.PersistentFlags().String(accessKeyIDFlag, "", accessKeyIDFlagUsage)
	rootCmd.PersistentFlags().String(accessKeySecretFlag, "", accessKeySecretFlagUsage)
	rootCmd.PersistentFlags().String(astAuthenticationURIFlag, "", astAuthenticationURIFlagUsage)

	// Bind the viper key ast_access_key_id to flag --key of the root command and
	// to the environment variable AST_ACCESS_KEY_ID so that it will be taken from environment variables first
	// and can be overridden by command flag --key
	_ = viper.BindPFlag(AccessKeyIDConfigKey, rootCmd.PersistentFlags().Lookup(accessKeyIDFlag))
	// Bind the viper key ast_access_key_secret to flag --secret of the root command and
	// to the environment variable AST_ACCESS_KEY_SECRET so that it will be taken from environment variables first
	// and can be overridden by command flag --secret
	_ = viper.BindPFlag(AccessKeySecretConfigKey, rootCmd.PersistentFlags().Lookup(accessKeySecretFlag))
	_ = viper.BindPFlag(AstAuthenticationURIConfigKey, rootCmd.PersistentFlags().Lookup(astAuthenticationURIFlag))

	scanCmd := NewScanCommand(scansWrapper, uploadsWrapper)
	projectCmd := NewProjectCommand(projectsWrapper)
	resultCmd := NewResultCommand(resultsWrapper)
	versionCmd := NewVersionCommand()

	rootCmd.AddCommand(scanCmd, projectCmd, resultCmd, versionCmd)
	rootCmd.SilenceUsage = true
	return rootCmd
}

func PrintIfVerbose(verbose bool, msg string) {
	if verbose {
		fmt.Println(msg)
	}
}

func getLimitAndOffset(cmd *cobra.Command) (limit, offset uint64) {
	verbose, _ := cmd.Flags().GetBool(verboseFlag)
	limit, _ = cmd.Flags().GetUint64(limitFlag)
	offset, _ = cmd.Flags().GetUint64(offsetFlag)
	PrintIfVerbose(verbose, fmt.Sprintf("%s: %d", limitFlag, limit))
	PrintIfVerbose(verbose, fmt.Sprintf("%s: %d", offsetFlag, offset))
	return
}
