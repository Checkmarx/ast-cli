package commands

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
	"github.com/pkg/errors"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const ErrorCodeFormat = "%s: CODE: %d, %s\n"

// NewAstCLI Return a Checkmarx One CLI root command to execute
func NewAstCLI(
	applicationsWrapper wrappers.ApplicationsWrapper,
	scansWrapper wrappers.ScansWrapper,
	resultsSbomWrapper wrappers.ResultsSbomWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper,
	codeBashingWrapper wrappers.CodeBashingWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	authWrapper wrappers.AuthWrapper,
	logsWrapper wrappers.LogsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	gitHubWrapper wrappers.GitHubWrapper,
	azureWrapper wrappers.AzureWrapper,
	bitBucketWrapper wrappers.BitBucketWrapper,
	bitBucketServerWrapper bitbucketserver.Wrapper,
	gitLabWrapper wrappers.GitLabWrapper,
	bflWrapper wrappers.BflWrapper,
	prWrapper wrappers.PRWrapper,
	learnMoreWrapper wrappers.LearnMoreWrapper,
	tenantWrapper wrappers.TenantConfigurationWrapper,
	jwtWrapper wrappers.JWTWrapper,
	scaRealTimeWrapper wrappers.ScaRealTimeWrapper,
	chatWrapper wrappers.ChatWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	policyWrapper wrappers.PolicyWrapper,
	sastMetadataWrapper wrappers.SastMetadataWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	byorWrapper wrappers.ByorWrapper,
) *cobra.Command {
	// Create the root
	rootCmd := &cobra.Command{
		Use:   "cx <command> <subcommand> [flags]",
		Short: "Checkmarx One CLI",
		Long:  "The Checkmarx One CLI is a fully functional Command Line Interface (CLI) that interacts with the Checkmarx One server.",
		Example: heredoc.Doc(
			`
			$ cx configure
			$ cx scan create -s . --project-name my_project_name
			$ cx scan list
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68620-checkmarx-one-cli-tool.html
			`,
			),
		},
	}

	// Load default flags
	rootCmd.PersistentFlags().Bool(params.DebugFlag, false, params.DebugUsage)
	rootCmd.PersistentFlags().String(params.AccessKeyIDFlag, "", params.AccessKeyIDFlagUsage)
	rootCmd.PersistentFlags().String(params.AccessKeySecretFlag, "", params.AccessKeySecretFlagUsage)
	rootCmd.PersistentFlags().Bool(params.InsecureFlag, false, params.InsecureFlagUsage)
	rootCmd.PersistentFlags().String(params.ProxyFlag, "", params.ProxyFlagUsage)
	rootCmd.PersistentFlags().Bool(params.IgnoreProxyFlag, false, params.IgnoreProxyFlagUsage)
	rootCmd.PersistentFlags().String(params.ProxyTypeFlag, "", params.ProxyTypeFlagUsage)
	rootCmd.PersistentFlags().String(params.NtlmProxyDomainFlag, "", params.NtlmProxyDomainFlagUsage)
	rootCmd.PersistentFlags().String(params.TimeoutFlag, "", params.TimeoutFlagUsage)
	rootCmd.PersistentFlags().String(params.BaseURIFlag, params.BaseURI, params.BaseURIFlagUsage)
	rootCmd.PersistentFlags().String(params.BaseAuthURIFlag, params.BaseIAMURI, params.BaseAuthURIFlagUsage)
	rootCmd.PersistentFlags().String(params.ProfileFlag, params.Profile, params.ProfileFlagUsage)
	rootCmd.PersistentFlags().String(params.AstAPIKeyFlag, "", params.AstAPIKeyUsage)
	rootCmd.PersistentFlags().String(params.AgentFlag, params.DefaultAgent, params.AgentFlagUsage)
	rootCmd.PersistentFlags().String(params.TenantFlag, params.Tenant, params.TenantFlagUsage)
	rootCmd.PersistentFlags().Uint(params.RetryFlag, params.RetryDefault, params.RetryUsage)
	rootCmd.PersistentFlags().Uint(params.RetryDelayFlag, params.RetryDelayDefault, params.RetryDelayUsage)

	rootCmd.PersistentFlags().Bool(params.ApikeyOverrideFlag, false, "")

	_ = rootCmd.PersistentFlags().MarkHidden(params.ApikeyOverrideFlag)

	// This monitors and traps situations where "extra/garbage" commands
	// are passed to Cobra.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		PrintConfiguration()
		// Need to check the __complete command to allow correct behavior of the autocomplete
		if len(args) > 0 && cmd.Name() != params.Help && cmd.Name() != "__complete" {
			_ = cmd.Help()
			os.Exit(0)
		}

		if requiredFeatureFlagsCheck(cmd) {
			err := wrappers.HandleFeatureFlags(featureFlagsWrapper)

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}
	// Link the environment variable to the CLI argument(s).
	_ = viper.BindPFlag(params.AccessKeyIDConfigKey, rootCmd.PersistentFlags().Lookup(params.AccessKeyIDFlag))
	_ = viper.BindPFlag(params.AccessKeySecretConfigKey, rootCmd.PersistentFlags().Lookup(params.AccessKeySecretFlag))
	_ = viper.BindPFlag(params.BaseURIKey, rootCmd.PersistentFlags().Lookup(params.BaseURIFlag))
	_ = viper.BindPFlag(params.TenantKey, rootCmd.PersistentFlags().Lookup(params.TenantFlag))
	_ = viper.BindPFlag(params.ProxyKey, rootCmd.PersistentFlags().Lookup(params.ProxyFlag))
	_ = viper.BindPFlag(params.ProxyTypeKey, rootCmd.PersistentFlags().Lookup(params.ProxyTypeFlag))
	_ = viper.BindPFlag(params.ProxyDomainKey, rootCmd.PersistentFlags().Lookup(params.NtlmProxyDomainFlag))
	_ = viper.BindPFlag(params.ClientTimeoutKey, rootCmd.PersistentFlags().Lookup(params.TimeoutFlag))
	_ = viper.BindPFlag(params.BaseAuthURIKey, rootCmd.PersistentFlags().Lookup(params.BaseAuthURIFlag))
	_ = viper.BindPFlag(params.AstAPIKey, rootCmd.PersistentFlags().Lookup(params.AstAPIKeyFlag))
	_ = viper.BindPFlag(params.AgentNameKey, rootCmd.PersistentFlags().Lookup(params.AgentFlag))
	_ = viper.BindPFlag(params.IgnoreProxyKey, rootCmd.PersistentFlags().Lookup(params.IgnoreProxyFlag))
	// Key here is the actual flag since it doesn't use an environment variable
	_ = viper.BindPFlag(params.DebugFlag, rootCmd.PersistentFlags().Lookup(params.DebugFlag))
	_ = viper.BindPFlag(params.InsecureFlag, rootCmd.PersistentFlags().Lookup(params.InsecureFlag))
	_ = viper.BindPFlag(params.RetryFlag, rootCmd.PersistentFlags().Lookup(params.RetryFlag))
	_ = viper.BindPFlag(params.RetryDelayFlag, rootCmd.PersistentFlags().Lookup(params.RetryDelayFlag))
	_ = viper.BindPFlag(params.ApikeyOverrideFlag, rootCmd.PersistentFlags().Lookup(params.ApikeyOverrideFlag))

	// Set help func
	rootCmd.SetHelpFunc(
		func(command *cobra.Command, args []string) {
			util.RootHelpFunc(command)
		},
	)

	// Create the CLI command structure
	scanCmd := NewScanCommand(
		applicationsWrapper,
		scansWrapper,
		resultsSbomWrapper,
		resultsPdfReportsWrapper,
		uploadsWrapper,
		resultsWrapper,
		projectsWrapper,
		logsWrapper,
		groupsWrapper,
		risksOverviewWrapper,
		jwtWrapper,
		scaRealTimeWrapper,
		policyWrapper,
		sastMetadataWrapper,
		accessManagementWrapper,
	)
	projectCmd := NewProjectCommand(applicationsWrapper, projectsWrapper, groupsWrapper, accessManagementWrapper)

	importCmd := NewImportCommand(projectsWrapper, uploadsWrapper, groupsWrapper, accessManagementWrapper, byorWrapper)

	resultsCmd := NewResultsCommand(
		resultsWrapper,
		scansWrapper,
		resultsSbomWrapper,
		resultsPdfReportsWrapper,
		codeBashingWrapper,
		bflWrapper,
		risksOverviewWrapper,
		policyWrapper,
	)

	versionCmd := util.NewVersionCommand()
	authCmd := NewAuthCommand(authWrapper)
	utilsCmd := util.NewUtilsCommand(
		gitHubWrapper,
		azureWrapper,
		bitBucketWrapper,
		bitBucketServerWrapper,
		gitLabWrapper,
		prWrapper,
		learnMoreWrapper,
		tenantWrapper,
		chatWrapper,
		policyWrapper,
		scansWrapper,
	)
	configCmd := util.NewConfigCommand()
	triageCmd := NewResultsPredicatesCommand(resultsPredicatesWrapper)

	chatCmd := NewChatCommand(chatWrapper, tenantWrapper)

	rootCmd.AddCommand(
		scanCmd,
		projectCmd,
		resultsCmd,
		triageCmd,
		versionCmd,
		authCmd,
		utilsCmd,
		configCmd,
		chatCmd,
	)

	if wrappers.FeatureFlags[featureFlagsConstants.ByorEnabled] {
		rootCmd.AddCommand(importCmd)
	}

	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	return rootCmd
}

func requiredFeatureFlagsCheck(cmd *cobra.Command) bool {
	for _, cmdFlag := range wrappers.FeatureFlagsBaseMap {
		if cmdFlag.CommandName == cmd.CommandPath() {
			return true
		}
	}

	return false
}

const configFormatString = "%30v: %s"

func PrintConfiguration() {
	logger.PrintfIfVerbose("CLI Version: %s", params.Version)
	logger.PrintIfVerbose("CLI Configuration:")
	for param := range util.Properties {
		logger.PrintIfVerbose(fmt.Sprintf(configFormatString, param, viper.GetString(param)))
	}
}

func getFilters(cmd *cobra.Command) (map[string]string, error) {
	filters, err := cmd.Flags().GetStringSlice(params.FilterFlag)
	if err != nil {
		return nil, err
	}
	allFilters := make(map[string]string)
	for _, filter := range filters {
		filterKeyVal := strings.Split(filter, "=")
		if len(filterKeyVal) != params.KeyValuePairSize {
			return nil, errors.Errorf("Invalid filters. Filters should be in a KEY=VALUE format")
		}

		allFilters[filterKeyVal[0]] = strings.Replace(
			filterKeyVal[1], ";", ",",
			strings.Count(filterKeyVal[1], ";"),
		)
	}
	return allFilters, nil
}

func addFormatFlagToMultipleCommands(commands []*cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	for _, c := range commands {
		addFormatFlag(c, defaultFormat, otherAvailableFormats...)
	}
}

func addFormatFlag(cmd *cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	cmd.PersistentFlags().String(
		params.FormatFlag, defaultFormat,
		fmt.Sprintf(params.FormatFlagUsageFormat, append(otherAvailableFormats, defaultFormat)),
	)
}

func addScanInfoFormatFlag(cmd *cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	cmd.PersistentFlags().String(
		params.ScanInfoFormatFlag, defaultFormat,
		fmt.Sprintf(params.FormatFlagUsageFormat, append(otherAvailableFormats, defaultFormat)),
	)
}

func addResultFormatFlag(cmd *cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	cmd.PersistentFlags().String(
		params.TargetFormatFlag, defaultFormat,
		fmt.Sprintf(params.FormatFlagUsageFormat, append(otherAvailableFormats, defaultFormat)),
	)
}

func markFlagAsRequired(cmd *cobra.Command, flag string) {
	err := cmd.MarkPersistentFlagRequired(flag)
	if err != nil {
		log.Fatal(err)
	}
}

func addScanIDFlag(cmd *cobra.Command, helpMsg string) {
	cmd.PersistentFlags().String(params.ScanIDFlag, "", helpMsg)
}

func addProjectIDFlag(cmd *cobra.Command, helpMsg string) {
	cmd.PersistentFlags().String(params.ProjectIDFlag, "", helpMsg)
}

func addQueryIDFlag(cmd *cobra.Command, helpMsg string) {
	cmd.PersistentFlags().String(params.QueryIDFlag, "", helpMsg)
}

func printByFormat(cmd *cobra.Command, view interface{}) error {
	f, _ := cmd.Flags().GetString(params.FormatFlag)
	return printer.Print(cmd.OutOrStdout(), view, f)
}

func printByScanInfoFormat(cmd *cobra.Command, view interface{}) error {
	f, _ := cmd.Flags().GetString(params.ScanInfoFormatFlag)
	return printer.Print(cmd.OutOrStdout(), view, f)
}
