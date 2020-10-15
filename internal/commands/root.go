package commands

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/pkg/errors"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	keyValuePairSize              = 2
	verboseFlag                   = "verbose"
	verboseFlagSh                 = "v"
	verboseUsage                  = "Verbose mode"
	sourcesFlag                   = "sources"
	sourcesFlagSh                 = "s"
	inputFlag                     = "input"
	inputFlagSh                   = "i"
	inputFileFlag                 = "input-file"
	inputFileFlagSh               = "f"
	accessKeyIDFlag               = "key"
	accessKeyIDFlagUsage          = "The access key ID"
	accessKeySecretFlag           = "secret"
	accessKeySecretFlagUsage      = "The access key secret"
	astAuthenticationURIFlag      = "auth-uri"
	astAuthenticationURIFlagUsage = "The authentication URI"
	insecureFlag                  = "insecure"
	insecureFlagUsage             = "Ignore TLS certificate validations"
	formatFlag                    = "format"
	formatFlagUsage               = "Format for the output. One of [json, list, table]"
	formatJSON                    = "json"
	formatList                    = "list"
	formatTable                   = "table"
	filterFlag                    = "filter"
	baseURIFlag                   = "base-uri"
	baseURIFlagUsage              = "The base system URI"
	queriesRepoNameFlag           = "name"
	queriesRepoNameSh             = "n"
	queriesRepoActivateFlag       = "activate"
	queriesRepoActivateSh         = "a"
	clientRolesFlag               = "roles"
	clientRolesSh                 = "r"
	clientDescriptionFlag         = "description"
	clientDescriptionSh           = "d"
	usernameFlag                  = "username"
	usernameSh                    = "u"
	passwordFlag                  = "password"
	passwordSh                    = "p"
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
	defaultConfigFileLocation string,
) *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "ast",
	}

	rootCmd.PersistentFlags().BoolP(verboseFlag, verboseFlagSh, false, verboseUsage)
	rootCmd.PersistentFlags().String(accessKeyIDFlag, "", accessKeyIDFlagUsage)
	rootCmd.PersistentFlags().String(accessKeySecretFlag, "", accessKeySecretFlagUsage)
	rootCmd.PersistentFlags().String(astAuthenticationURIFlag, "", astAuthenticationURIFlagUsage)
	rootCmd.PersistentFlags().Bool(insecureFlag, false, insecureFlagUsage)
	// Default value is table. Should we? or JSON?
	rootCmd.PersistentFlags().String(formatFlag, formatTable, formatFlagUsage)
	rootCmd.PersistentFlags().String(baseURIFlag, params.BaseURI, baseURIFlagUsage)

	// Bind the viper key ast_access_key_id to flag --key of the root command and
	// to the environment variable AST_ACCESS_KEY_ID so that it will be taken from environment variables first
	// and can be overridden by command flag --key
	_ = viper.BindPFlag(params.AccessKeyIDConfigKey, rootCmd.PersistentFlags().Lookup(accessKeyIDFlag))
	_ = viper.BindPFlag(params.AccessKeySecretConfigKey, rootCmd.PersistentFlags().Lookup(accessKeySecretFlag))
	_ = viper.BindPFlag(params.AstAuthenticationURIConfigKey, rootCmd.PersistentFlags().Lookup(astAuthenticationURIFlag))
	_ = viper.BindPFlag(params.BaseURIKey, rootCmd.PersistentFlags().Lookup(baseURIFlag))
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
	singleNodeCommand := NewSingleNodeCommand(healthCheckWrapper, defaultConfigFileLocation)
	rmCmd := NewSastResourcesCommand(rmWrapper)
	queriesCmd := NewQueryCommand(queriesWrapper, uploadsWrapper)
	authCmd := NewAuthCommand(authWrapper)

	rootCmd.AddCommand(scanCmd,
		projectCmd,
		resultCmd,
		versionCmd,
		clusterCmd,
		appCmd,
		singleNodeCommand,
		bflCmd,
		rmCmd,
		queriesCmd,
		authCmd,
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

func runBashCommand(name string, envs []string, arg ...string) (*bytes.Buffer, *bytes.Buffer, error) { // nolint
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command(name, arg...)
	cmd.Env = envs
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Run()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Error running command %s", name)
	}
	return &stdoutBuf, &stderrBuf, nil
}
