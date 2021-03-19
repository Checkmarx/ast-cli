package commands

import (
	"fmt"
	"strings"

	"github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/pkg/errors"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	keyValuePairSize               = 2
	verboseFlag                    = "verbose"
	verboseFlagSh                  = "v"
	verboseUsage                   = "Verbose mode"
	sourcesFlag                    = "sources"
	sourcesFlagSh                  = "s"
	sourceDirFlag                  = "directory"
	sourceDirFlagSh                = "d"
	sourceDirFilterFlag            = "filter"
	sourceDirFilterFlagSh          = "f"
	mainBranchFlag                 = "branch"
	projectName                    = "project-name"
	projectSourceType              = "project-source-type"
	projectType                    = "project-type"
	incremental                    = "incremental"
	presetName                     = "preset-name"
	accessKeyIDFlag                = "client-id"
	accessKeyIDFlagUsage           = "The access key ID"
	accessKeySecretFlag            = "secret"
	accessKeySecretFlagUsage       = "The access key secret"
	astAuthenticationPathFlag      = "auth-path"
	astAuthenticationPathFlagUsage = "The authentication path"
	insecureFlag                   = "insecure"
	insecureFlagUsage              = "Ignore TLS certificate validations"
	formatFlag                     = "format"
	formatFlagUsageFormat          = "Format for the output. One of %s"
	formatJSON                     = "json"
	formatList                     = "list"
	formatTable                    = "table"
	filterFlag                     = "filter"
	baseURIFlag                    = "base-uri"
	proxyFlag                      = "proxy"
	proxyFlagUsage                 = "Proxy server to send communication through"
	baseURIFlagUsage               = "The base system URI"
	baseIAMURIFlag                 = "base-auth-uri"
	baseIAMURIFlagUsage            = "The base system IAM URI"
	astTokenFlag                   = "token"
	astTokenUsage                  = "The token to login to AST with"
	repoUrlFlag                    = "repo-url"
	queriesRepoNameFlag            = "name"
	queriesRepoNameSh              = "n"
	queriesRepoActivateFlag        = "activate"
	queriesRepoActivateSh          = "a"
	clientRolesFlag                = "roles"
	clientRolesSh                  = "r"
	clientDescriptionFlag          = "description"
	clientDescriptionSh            = "d"
	usernameFlag                   = "username"
	usernameSh                     = "u"
	passwordFlag                   = "password"
	passwordSh                     = "p"
	profileFlag                    = "profile"
	profileFlagUsage               = "The default configuration profile"
)

// Return an AST CLI root command to execute
func NewAstCLI(
	scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	bflWrapper wrappers.BFLWrapper,
	rmWrapper wrappers.SastRmWrapper,
	healthCheckWrapper wrappers.HealthCheckWrapper,
	queriesWrapper wrappers.QueriesWrapper,
	authWrapper wrappers.AuthWrapper,
	ssiWrapper wrappers.SastMetadataWrapper,
	logsWrapper wrappers.LogsWrapper,
) *cobra.Command {
	// Create the root
	rootCmd := &cobra.Command{
		Use: "cx",
	}

	// Load default flags
	rootCmd.PersistentFlags().BoolP(verboseFlag, verboseFlagSh, false, verboseUsage)
	rootCmd.PersistentFlags().String(accessKeyIDFlag, "", accessKeyIDFlagUsage)
	rootCmd.PersistentFlags().String(accessKeySecretFlag, "", accessKeySecretFlagUsage)
	rootCmd.PersistentFlags().String(astAuthenticationPathFlag, "", astAuthenticationPathFlagUsage)
	rootCmd.PersistentFlags().Bool(insecureFlag, false, insecureFlagUsage)
	rootCmd.PersistentFlags().String(proxyFlag, "", proxyFlagUsage)
	rootCmd.PersistentFlags().String(baseURIFlag, params.BaseURI, baseURIFlagUsage)
	rootCmd.PersistentFlags().String(baseIAMURIFlag, params.BaseIAMURI, baseIAMURIFlagUsage)
	rootCmd.PersistentFlags().String(profileFlag, params.Profile, profileFlagUsage)
	rootCmd.PersistentFlags().String(astTokenFlag, params.BaseURI, astTokenUsage)

	// Bind the viper key ast_access_key_id to flag --key of the root command and
	// to the environment variable AST_ACCESS_KEY_ID so that it will be taken from environment variables first
	// and can be overridden by command flag --key
	_ = viper.BindPFlag(params.AccessKeyIDConfigKey, rootCmd.PersistentFlags().Lookup(accessKeyIDFlag))
	_ = viper.BindPFlag(params.AccessKeySecretConfigKey, rootCmd.PersistentFlags().Lookup(accessKeySecretFlag))
	_ = viper.BindPFlag(params.AstAuthenticationPathConfigKey, rootCmd.PersistentFlags().Lookup(astAuthenticationPathFlag))
	_ = viper.BindPFlag(params.BaseURIKey, rootCmd.PersistentFlags().Lookup(baseURIFlag))
	_ = viper.BindPFlag(params.ProxyKey, rootCmd.PersistentFlags().Lookup(proxyFlag))
	_ = viper.BindPFlag(params.BaseIAMURIKey, rootCmd.PersistentFlags().Lookup(baseIAMURIFlag))
	_ = viper.BindPFlag(params.AstTokenKey, rootCmd.PersistentFlags().Lookup(astTokenFlag))
	// Key here is the actual flag since it doesn't use an environment variable
	_ = viper.BindPFlag(verboseFlag, rootCmd.PersistentFlags().Lookup(verboseFlag))
	_ = viper.BindPFlag(insecureFlag, rootCmd.PersistentFlags().Lookup(insecureFlag))

	// Create the CLI command structure
	scanCmd := NewScanCommand(scansWrapper, uploadsWrapper)
	projectCmd := NewProjectCommand(projectsWrapper)
	resultCmd := NewResultCommand(resultsWrapper)
	bflCmd := NewBFLCommand(bflWrapper)
	versionCmd := NewVersionCommand()
	authCmd := NewAuthCommand(authWrapper)
	utilsCmd := NewUtilsCommand(healthCheckWrapper, ssiWrapper, rmWrapper, logsWrapper, queriesWrapper, uploadsWrapper)
	configCmd := NewConfigCommand()

	rootCmd.AddCommand(scanCmd,
		projectCmd,
		resultCmd,
		versionCmd,
		bflCmd,
		authCmd,
		utilsCmd,
		configCmd,
	)
	rootCmd.SilenceUsage = true
	return rootCmd
}

func PrintIfVerbose(msg string) {
	if viper.GetBool(verboseFlag) {
		writeToStandardOutput(msg)
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
		if len(filterKeyVal) != keyValuePairSize {
			return nil, errors.Errorf("Invalid filters. Filters should be in a KEY=VALUE format")
		}

		allFilters[filterKeyVal[0]] = strings.Replace(
			filterKeyVal[1], ";", ",",
			strings.Count(filterKeyVal[1], ";"))
	}
	return allFilters, nil
}

func addFormatFlagToMultipleCommands(cmds []*cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	for _, c := range cmds {
		addFormatFlag(c, defaultFormat, otherAvailableFormats...)
	}
}

func addFormatFlag(cmd *cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	cmd.PersistentFlags().String(formatFlag, defaultFormat,
		fmt.Sprintf(formatFlagUsageFormat, append(otherAvailableFormats, defaultFormat)))
}

func printByFormat(cmd *cobra.Command, view interface{}) error {
	f, _ := cmd.Flags().GetString(formatFlag)
	return Print(cmd.OutOrStdout(), view, f)
}
