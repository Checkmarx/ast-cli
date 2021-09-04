package commands

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarxDev/ast-cli/internal/commands/util"
	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/pkg/errors"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var singleNodeLogger = log.New(deploymentLogWriter{}, "", 0)

const (
	KeyValuePairSize         = 2
	VerboseFlag              = "verbose"
	VerboseFlagSh            = "v"
	VerboseUsage             = "Verbose mode"
	SourcesFlag              = "file-source"
	SourcesFlagSh            = "s"
	TenantFlag               = "tenant"
	TenantFlagUsage          = "Checkmarx tenant"
	AgentFlag                = "agent"
	AgentFlagUsage           = "Scan origin name"
	WaitFlag                 = "nowait"
	WaitDelayFlag            = "wait-delay"
	WaitDelayDefault         = 5
	SourceDirFilterFlag      = "file-filter"
	SourceDirFilterFlagSh    = "f"
	IncludeFilterFlag        = "file-include"
	IncludeFilterFlagSh      = "i"
	ProjectIDFlag            = "project-id"
	BranchFlag               = "branch"
	BranchFlagSh             = "b"
	ScanIDFlag               = "scan-id"
	BranchFlagUsage          = "Branch to scan"
	MainBranchFlag           = "branch"
	ProjectName              = "project-name"
	ScanTypes                = "scan-types"
	TagList                  = "tags"
	GroupList                = "groups"
	IncrementalSast          = "sast-incremental"
	PresetName               = "sast-preset-name"
	ScaResolver              = "sca-resolver"
	AccessKeyIDFlag          = "client-id"
	AccessKeySecretFlag      = "client-secret"
	AccessKeyIDFlagUsage     = "The OAuth2 client ID"
	AccessKeySecretFlagUsage = "The OAuth2 client secret"
	InsecureFlag             = "insecure"
	InsecureFlagUsage        = "Ignore TLS certificate validations"
	FormatFlag               = "format"
	FormatFlagUsageFormat    = "Format for the output. One of %s"
	FilterFlag               = "filter"
	BaseURIFlag              = "base-uri"
	ProxyFlag                = "proxy"
	ProxyFlagUsage           = "Proxy server to send communication through"
	ProxyTypeFlag            = "proxy-auth-type"
	ProxyTypeFlagUsage       = "Proxy authentication type, (basic or ntlm)"
	NtlmProxyDomainFlag      = "proxy-ntlm-domain"
	NtlmProxyDomainFlagUsage = "Window domain when using NTLM proxy"
	BaseURIFlagUsage         = "The base system URI"
	BaseAuthURIFlag          = "base-auth-uri"
	BaseAuthURIFlagUsage     = "The base system IAM URI"
	AstAPIKeyFlag            = "apikey"
	AstAPIKeyUsage           = "The API Key to login to AST"
	RepoURLFlag              = "repo-url"
	ClientRolesFlag          = "roles"
	ClientRolesSh            = "r"
	ClientDescriptionFlag    = "description"
	ClientDescriptionSh      = "d"
	UsernameFlag             = "username"
	UsernameSh               = "u"
	PasswordFlag             = "password"
	PasswordSh               = "p"
	ProfileFlag              = "profile"
	ProfileFlagUsage         = "The default configuration profile"
	Help                     = "help"
	TargetFlag               = "output-name"
	TargetPathFlag           = "output-path"
	TargetFormatFlag         = "report-format"
)

// NewAstCLI Return an AST CLI root command to execute
func NewAstCLI(
	scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	authWrapper wrappers.AuthWrapper,
) *cobra.Command {
	// Create the root
	rootCmd := &cobra.Command{
		Use:   "cx <command> <subcommand> [flags]",
		Short: "Checkmarx AST CLI",
		Long:  "The AST CLI is a fully functional Command Line Interface (CLI) that interacts with the Checkmarx CxAST server.",
		Example: heredoc.Doc(`
			$ cx configure
			$ cx scan create -s . --project-name my_project_name
			$ cx scan list
		`),
		Annotations: map[string]string{
			"utils:env": heredoc.Doc(`
				See 'cx utils env' for  the list of supported environment variables.
			`),
			"command:doc": heredoc.Doc(`
				https://checkmarx.atlassian.net/wiki/x/MYDCkQ
			`),
		},
	}

	// Load default flags
	rootCmd.PersistentFlags().BoolP(VerboseFlag, VerboseFlagSh, false, VerboseUsage)
	rootCmd.PersistentFlags().String(AccessKeyIDFlag, "", AccessKeyIDFlagUsage)
	rootCmd.PersistentFlags().String(AccessKeySecretFlag, "", AccessKeySecretFlagUsage)
	rootCmd.PersistentFlags().Bool(InsecureFlag, false, InsecureFlagUsage)
	rootCmd.PersistentFlags().String(ProxyFlag, "", ProxyFlagUsage)
	rootCmd.PersistentFlags().String(ProxyTypeFlag, "", ProxyTypeFlagUsage)
	rootCmd.PersistentFlags().String(NtlmProxyDomainFlag, "", NtlmProxyDomainFlagUsage)
	rootCmd.PersistentFlags().String(BaseURIFlag, params.BaseURI, BaseURIFlagUsage)
	rootCmd.PersistentFlags().String(BaseAuthURIFlag, params.BaseIAMURI, BaseAuthURIFlagUsage)
	rootCmd.PersistentFlags().String(ProfileFlag, params.Profile, ProfileFlagUsage)
	rootCmd.PersistentFlags().String(AstAPIKeyFlag, "", AstAPIKeyUsage)
	rootCmd.PersistentFlags().String(AgentFlag, params.AgentFlag, AgentFlagUsage)
	rootCmd.PersistentFlags().String(TenantFlag, params.Tenant, TenantFlagUsage)

	// This monitors and traps situations where "extra/garbage" commands
	// are passed to Cobra.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if len(args) > 0 && cmd.Name() != Help {
			_ = cmd.Help()
			os.Exit(0)
		}
	}

	// Link the environment variable to the CLI argument(s).
	_ = viper.BindPFlag(params.AccessKeyIDConfigKey, rootCmd.PersistentFlags().Lookup(AccessKeyIDFlag))
	_ = viper.BindPFlag(params.AccessKeySecretConfigKey, rootCmd.PersistentFlags().Lookup(AccessKeySecretFlag))
	_ = viper.BindPFlag(params.BaseURIKey, rootCmd.PersistentFlags().Lookup(BaseURIFlag))
	_ = viper.BindPFlag(params.TenantKey, rootCmd.PersistentFlags().Lookup(TenantFlag))
	_ = viper.BindPFlag(params.ProxyKey, rootCmd.PersistentFlags().Lookup(ProxyFlag))
	_ = viper.BindPFlag(params.ProxyTypeKey, rootCmd.PersistentFlags().Lookup(ProxyTypeFlag))
	_ = viper.BindPFlag(params.ProxyDomainKey, rootCmd.PersistentFlags().Lookup(NtlmProxyDomainFlag))
	_ = viper.BindPFlag(params.BaseAuthURIKey, rootCmd.PersistentFlags().Lookup(BaseAuthURIFlag))
	_ = viper.BindPFlag(params.AstAPIKey, rootCmd.PersistentFlags().Lookup(AstAPIKeyFlag))
	_ = viper.BindPFlag(params.AgentNameKey, rootCmd.PersistentFlags().Lookup(AgentFlag))
	// Key here is the actual flag since it doesn't use an environment variable
	_ = viper.BindPFlag(VerboseFlag, rootCmd.PersistentFlags().Lookup(VerboseFlag))
	_ = viper.BindPFlag(InsecureFlag, rootCmd.PersistentFlags().Lookup(InsecureFlag))

	// Set help func
	rootCmd.SetHelpFunc(func(command *cobra.Command, args []string) {
		util.RootHelpFunc(command)
	})

	// Create the CLI command structure
	scanCmd := NewScanCommand(scansWrapper, uploadsWrapper, resultsWrapper)
	projectCmd := NewProjectCommand(projectsWrapper)
	resultCmd := NewResultCommand(resultsWrapper)
	versionCmd := util.NewVersionCommand()
	authCmd := NewAuthCommand(authWrapper)
	utilsCmd := util.NewUtilsCommand()
	configCmd := util.NewConfigCommand()

	rootCmd.AddCommand(scanCmd,
		projectCmd,
		resultCmd,
		versionCmd,
		authCmd,
		utilsCmd,
		configCmd,
	)

	rootCmd.SilenceUsage = true
	return rootCmd
}

type deploymentLogWriter struct {
}

func (writer deploymentLogWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().Format(time.RFC3339) + " " + string(bytes))
}

func PrintIfVerbose(msg string) {
	if viper.GetBool(VerboseFlag) {
		singleNodeLogger.Println(msg)
	}
}

func getFilters(cmd *cobra.Command) (map[string]string, error) {
	filters, err := cmd.Flags().GetStringSlice(FilterFlag)
	if err != nil {
		return nil, err
	}
	allFilters := make(map[string]string)
	for _, filter := range filters {
		filterKeyVal := strings.Split(filter, "=")
		if len(filterKeyVal) != KeyValuePairSize {
			return nil, errors.Errorf("Invalid filters. Filters should be in a KEY=VALUE format")
		}

		allFilters[filterKeyVal[0]] = strings.Replace(
			filterKeyVal[1], ";", ",",
			strings.Count(filterKeyVal[1], ";"))
	}
	return allFilters, nil
}

func addFormatFlagToMultipleCommands(commands []*cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	for _, c := range commands {
		addFormatFlag(c, defaultFormat, otherAvailableFormats...)
	}
}

func addFormatFlag(cmd *cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	cmd.PersistentFlags().String(FormatFlag, defaultFormat,
		fmt.Sprintf(FormatFlagUsageFormat, append(otherAvailableFormats, defaultFormat)))
}

func addResultFormatFlag(cmd *cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	cmd.PersistentFlags().String(TargetFormatFlag, defaultFormat,
		fmt.Sprintf(FormatFlagUsageFormat, append(otherAvailableFormats, defaultFormat)))
}

func addScanIDFlag(cmd *cobra.Command, helpMsg string) {
	cmd.PersistentFlags().String(ScanIDFlag, "", helpMsg)
}

func addProjectIDFlag(cmd *cobra.Command, helpMsg string) {
	cmd.PersistentFlags().String(ProjectIDFlag, "", helpMsg)
}

func printByFormat(cmd *cobra.Command, view interface{}) error {
	f, _ := cmd.Flags().GetString(FormatFlag)
	return util.Print(cmd.OutOrStdout(), view, f)
}
