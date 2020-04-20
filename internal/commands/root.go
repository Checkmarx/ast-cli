package commands

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	keyValueSizeBecauseOfLinter   = 2
	verboseFlag                   = "verbose"
	verboseFlagSh                 = "v"
	verboseUsage                  = "Verbose mode"
	sourcesFlag                   = "sources"
	sourcesFlagSh                 = "s"
	inputFlag                     = "input"
	inputFlagSh                   = "i"
	inputFileFlag                 = "input-file"
	inputFileFlagSh               = "f"
	AccessKeyIDEnv                = "AST_ACCESS_KEY_ID"
	accessKeyIDFlag               = "key"
	accessKeyIDFlagUsage          = "The access key ID for AST"
	AccessKeySecretEnv            = "AST_ACCESS_KEY_SECRET"
	accessKeySecretFlag           = "secret"
	accessKeySecretFlagUsage      = "The access key secret for AST"
	AstAuthenticationURIEnv       = "AST_AUTHENTICATION_URI"
	astAuthenticationURIFlag      = "auth-uri"
	astAuthenticationURIFlagUsage = "The authentication URI for AST"
	insecureFlag                  = "insecure"
	insecureFlagUsage             = "Ignore TLS certificate validations"
	formatFlag                    = "format"
	formatFlagUsage               = "Format for the output. One of [json, pretty]. Default is JSON"
	formatJSON                    = "json"
	formatPretty                  = "pretty"
	filterFlag                    = "filter"
)

var (
	AccessKeyIDConfigKey          = strings.ToLower(AccessKeyIDEnv)
	AccessKeySecretConfigKey      = strings.ToLower(AccessKeySecretEnv)
	AstAuthenticationURIConfigKey = strings.ToLower(AstAuthenticationURIEnv)
)

// Return an AST CLI root command to execute
func NewAstCLI(
	scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	bflWrapper wrappers.BFLWrapper) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ast",
		Short: "A CLI wrapping Checkmarx AST APIs",
	}

	rootCmd.PersistentFlags().BoolP(verboseFlag, verboseFlagSh, false, verboseUsage)
	rootCmd.PersistentFlags().String(accessKeyIDFlag, "", accessKeyIDFlagUsage)
	rootCmd.PersistentFlags().String(accessKeySecretFlag, "", accessKeySecretFlagUsage)
	rootCmd.PersistentFlags().String(astAuthenticationURIFlag, "", astAuthenticationURIFlagUsage)
	rootCmd.PersistentFlags().Bool(insecureFlag, false, insecureFlagUsage)
	rootCmd.PersistentFlags().String(formatFlag, formatJSON, formatFlagUsage)

	// Bind the viper key ast_access_key_id to flag --key of the root command and
	// to the environment variable AST_ACCESS_KEY_ID so that it will be taken from environment variables first
	// and can be overridden by command flag --key
	_ = viper.BindPFlag(AccessKeyIDConfigKey, rootCmd.PersistentFlags().Lookup(accessKeyIDFlag))
	_ = viper.BindPFlag(AccessKeySecretConfigKey, rootCmd.PersistentFlags().Lookup(accessKeySecretFlag))
	_ = viper.BindPFlag(AstAuthenticationURIConfigKey, rootCmd.PersistentFlags().Lookup(astAuthenticationURIFlag))
	// Key here is the actual flag since it doesn't use an environment variable
	_ = viper.BindPFlag(verboseFlag, rootCmd.PersistentFlags().Lookup(verboseFlag))
	_ = viper.BindPFlag(insecureFlag, rootCmd.PersistentFlags().Lookup(insecureFlag))
	_ = viper.BindPFlag(formatFlag, rootCmd.PersistentFlags().Lookup(formatFlag))

	scanCmd := NewScanCommand(scansWrapper, uploadsWrapper)
	projectCmd := NewProjectCommand(projectsWrapper)
	resultCmd := NewResultCommand(resultsWrapper)
	bflCmd := NewBFLCommand(bflWrapper)
	versionCmd := NewVersionCommand()
	clusterCmd := NewClusterCommand()
	appCmd := NewAppCommand()

	rootCmd.AddCommand(scanCmd, projectCmd, resultCmd, versionCmd, clusterCmd, appCmd, bflCmd)
	rootCmd.SilenceUsage = true
	return rootCmd
}

func PrintIfVerbose(msg string) {
	if viper.GetBool(verboseFlag) {
		fmt.Println(msg)
	}
}

func getFilters(cmd *cobra.Command) (map[string]string, error) {
	filters, err := cmd.Flags().GetStringSlice(filterFlag)
	if err != nil {
		return nil, err
	}
	allFilters := make(map[string]string)
	for _, filter := range filters {
		filterKeyVal := strings.Split(filter, "=")
		if len(filterKeyVal) != keyValueSizeBecauseOfLinter {
			return nil, errors.Errorf("Invalid filters. Filters should be in a KEY=VALUE format")
		}
		allFilters[filterKeyVal[0]] = filterKeyVal[1]
	}
	return allFilters, nil
}

func IsJSONFormat() bool {
	return strings.EqualFold(viper.GetString(formatFlag), formatJSON)
}
func IsPrettyFormat() bool {
	return strings.EqualFold(viper.GetString(formatFlag), formatPretty)
}

func ValidateFormat() error {
	if !(IsPrettyFormat() || IsJSONFormat()) {
		return errors.Errorf("Invalid format %s", viper.GetString(formatFlag))
	}
	return nil
}
