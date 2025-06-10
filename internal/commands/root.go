package commands

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
	"github.com/pkg/errors"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewAstCLI Return a Checkmarx One CLI root command to execute
func NewAstCLI(
	applicationsWrapper wrappers.ApplicationsWrapper,
	scansWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper,
	customStatesWrapper wrappers.CustomStatesWrapper,
	codeBashingWrapper wrappers.CodeBashingWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	riskManagementWrapper wrappers.RiskManagementWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
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
	containerResolverWrapper wrappers.ContainerResolverWrapper,
	realTimeWrapper wrappers.RealtimeScannerWrapper,
) *cobra.Command {
	// Create the root
	rootCmd := &cobra.Command{
		Use:   "cx <command> <subcommand> [flags]",
		Short: "Checkmarx One CLI",
		Long:  "The Checkmarx One CLI is a fully functional Command Line Interface (CLI) that interacts with the Checkmarx One server",
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
	_ = viper.BindPFlag(params.OriginKey, rootCmd.PersistentFlags().Lookup(params.OriginFlag))
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
		exportWrapper,
		resultsPdfReportsWrapper,
		uploadsWrapper,
		resultsWrapper,
		projectsWrapper,
		logsWrapper,
		groupsWrapper,
		risksOverviewWrapper,
		scsScanOverviewWrapper,
		jwtWrapper,
		scaRealTimeWrapper,
		policyWrapper,
		sastMetadataWrapper,
		accessManagementWrapper,
		featureFlagsWrapper,
		containerResolverWrapper,
		realTimeWrapper,
	)
	projectCmd := NewProjectCommand(applicationsWrapper, projectsWrapper, groupsWrapper, accessManagementWrapper, featureFlagsWrapper)

	resultsCmd := NewResultsCommand(
		resultsWrapper,
		scansWrapper,
		exportWrapper,
		resultsPdfReportsWrapper,
		codeBashingWrapper,
		bflWrapper,
		risksOverviewWrapper,
		riskManagementWrapper,
		scsScanOverviewWrapper,
		policyWrapper,
		featureFlagsWrapper,
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
		projectsWrapper,
		uploadsWrapper,
		groupsWrapper,
		accessManagementWrapper,
		applicationsWrapper,
		byorWrapper,
		featureFlagsWrapper,
	)

	configCmd := util.NewConfigCommand()
	triageCmd := NewResultsPredicatesCommand(resultsPredicatesWrapper, featureFlagsWrapper, customStatesWrapper)

	chatCmd := NewChatCommand(chatWrapper, tenantWrapper)
	hooksCmd := NewHooksCommand(jwtWrapper)

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
		hooksCmd,
	)

	rootCmd.SilenceUsage = true
	return rootCmd
}

const configFormatString = "%30v: %s"

var extraFilter = map[string]map[string]string{
	"state": {
		"exclude_not_exploitable": "TO_VERIFY;PROPOSED_NOT_EXPLOITABLE;CONFIRMED;URGENT",
		"EXCLUDE_NOT_EXPLOITABLE": "TO_VERIFY;PROPOSED_NOT_EXPLOITABLE;CONFIRMED;URGENT",
	},
	"severity": {},
	"status":   {},
	"sort":     {},
}

func PrintConfiguration() {
	logger.PrintfIfVerbose("CLI Version: %s", params.Version)
	logger.PrintIfVerbose("CLI Configuration:")
	for param := range util.Properties {
		logger.PrintIfVerbose(fmt.Sprintf(configFormatString, param, viper.GetString(param)))
	}
}

func getFilters(cmd *cobra.Command) (map[string]string, error) {
	filters, _ := cmd.Flags().GetStringSlice(params.FilterFlag)
	allFilters := make(map[string]string)
	for _, filter := range filters {
		filterKeyVal := strings.Split(filter, "=")
		if len(filterKeyVal) != params.KeyValuePairSize {
			return nil, errors.New("Invalid filters. Filters should be in a KEY=VALUE format")
		}
		filterKeyVal = validateExtraFilters(filterKeyVal)
		allFilters[filterKeyVal[0]] = strings.Replace(
			filterKeyVal[1], ";", ",",
			strings.Count(filterKeyVal[1], ";"),
		)
	}
	return allFilters, nil
}

func validateExtraFilters(filterKeyVal []string) []string {
	// Add support for state = exclude-not-exploitable, will replace all values of filter flag state to "TO_VERIFY;PROPOSED_NOT_EXPLOITABLE;CONFIRMED;URGENT"
	if extraFilter[filterKeyVal[0]] != nil {
		for privateFilter, value := range extraFilter[filterKeyVal[0]] {
			if strings.Contains(filterKeyVal[1], privateFilter) {
				logger.PrintfIfVerbose("Set filter from extra filters: [%s,%s]", filterKeyVal[0], value)
				filterKeyVal[1] = strings.Replace(filterKeyVal[1], filterKeyVal[1], value, -1)
			}
		}
	}
	return filterKeyVal
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
