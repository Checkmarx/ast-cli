package commands

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/checkmarx/ast-cli/internal/commands/asca"
	"github.com/checkmarx/ast-cli/internal/commands/scarealtime"
	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/constants"
	exitCodes "github.com/checkmarx/ast-cli/internal/constants/exit-codes"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/MakeNowJust/heredoc"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/mssola/user_agent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	failedCreating    = "Failed creating a scan"
	failedGetting     = "Failed showing a scan"
	failedGettingTags = "Failed getting tags"
	failedDeleting    = "Failed deleting a scan"
	failedCanceling   = "Failed canceling a scan"
	failedGettingAll  = "Failed listing"
	thresholdLog      = "%s: Limit = %d, Current = %v"
	thresholdMsgLog   = "Threshold check finished with status %s : %s"
	mbBytes           = 1024.0 * 1024.0
	notExploitable    = "NOT_EXPLOITABLE"
	ignored           = "IGNORED"

	git                                     = "git"
	invalidSSHSource                        = "provided source does not need a key. Make sure you are defining the right source or remove the flag --ssh-key"
	errorUnzippingFile                      = "an error occurred while unzipping file. Reason: "
	containerRun                            = "run"
	containerVolumeFlag                     = "-v"
	containerNameFlag                       = "--name"
	containerRemove                         = "--rm"
	containerImage                          = "checkmarx/kics:v2.1.11"
	containerScan                           = "scan"
	containerScanPathFlag                   = "-p"
	containerScanPath                       = "/path"
	containerScanOutputFlag                 = "-o"
	containerScanOutput                     = "/path"
	containerScanFormatFlag                 = "--report-formats"
	containerScanFormatOutput               = "json"
	containerStarting                       = "Starting kics container"
	containerFormatInfo                     = "The report format and output path cannot be overridden."
	containerFolderRemoving                 = "Removing folder in temp"
	containerCreateFolderError              = "Error creating temporary directory"
	containerWriteFolderError               = " Error writing file to temporary directory"
	containerFileSourceMissing              = "--file is required for kics-realtime command"
	containerFileSourceIncompatible         = ". Provided file is not supported by kics"
	containerFileSourceError                = " Error reading file"
	containerResultsFileFormat              = "%s/results.json"
	containerVolumeFormat                   = "%s:/path"
	containerTempDirPattern                 = "kics"
	kicsContainerPrefixName                 = "cli-kics-realtime-"
	cleanupMaxRetries                       = 3
	cleanupRetryWaitSeconds                 = 15
	DanglingSymlinkError                    = "Skipping dangling symbolic link"
	configFilterKey                         = "filter"
	configFilterPlatforms                   = "platforms"
	configIncremental                       = "incremental"
	configFastScan                          = "fastScanMode"
	configLightQueries                      = "lightQueries"
	configRecommendedExclusions             = "recommendedExclusions"
	configPresetName                        = "presetName"
	configPresetID                          = "presetId"
	configEngineVerbose                     = "engineVerbose"
	configLanguageMode                      = "languageMode"
	ConfigContainersFilesFilterKey          = "filesFilter"
	ConfigContainersImagesFilterKey         = "imagesFilter"
	ConfigContainersPackagesFilterKey       = "packagesFilter"
	ConfigContainersNonFinalStagesFilterKey = "nonFinalStagesFilter"
	ConfigUserCustomImagesKey               = "userCustomImages"
	resultsMapValue                         = "value"
	resultsMapType                          = "type"
	trueString                              = "true"
	configTwoms                             = "2ms"
	falseString                             = "false"
	maxPollingWaitTime                      = 60
	engineNotAllowed                        = "It looks like the \"%s\" scan type does not exist or you are trying to run a scan without the \"%s\" package license." +
		"\nTo use this feature, you would need to purchase a license." +
		"\nPlease contact our support team for assistance if you believe you have already purchased a license." +
		"\nLicensed packages: %s"
	containerResolutionFileName = "containers-resolution.json"
	directoryCreationPrefix     = "cx-"
	ScsScoreCardType            = "scorecard"
	ScsSecretDetectionType      = "secret-detection"
	ScsRepoRequiredMsg          = "SCS scan failed to start: Scorecard scan is missing required flags, please include in the ast-cli arguments: " +
		"--scs-repo-url your_repo_url --scs-repo-token your_repo_token"
	ScsRepoWarningMsg = "SCS scan warning: Unable to start Scorecard scan due to missing required flags, please include in the ast-cli arguments: " +
		"--scs-repo-url your_repo_url --scs-repo-token your_repo_token"
	ScsScorecardUnsupportedHostWarningMsg = "SCS scan warning: Unable to run Scorecard scanner due to unsupported repo host. Currently, Scorecard can only run on GitHub Cloud repos."

	jsonExt             = ".json"
	xmlExt              = ".xml"
	sbomScanTypeErrMsg  = "The --sbom-only flag can only be used when the scan type is sca"
	BranchPrimaryPrefix = "--branch-primary="
)

var (
	scaResolverResultsFile  = ""
	actualScanTypes         = "sast,kics,sca,scs"
	filterScanListFlagUsage = fmt.Sprintf(
		"Filter the list of scans. Use ';' as the delimeter for arrays. Available filters are: %s",
		strings.Join(
			[]string{
				commonParams.LimitQueryParam,
				commonParams.OffsetQueryParam,
				commonParams.ScanIDsQueryParam,
				commonParams.TagsKeyQueryParam,
				commonParams.TagsValueQueryParam,
				commonParams.StatusesQueryParam,
				commonParams.ProjectIDQueryParam,
				commonParams.FromDateQueryParam,
				commonParams.ToDateQueryParam,
			}, ",",
		),
	)
	aditionalParameters []string
	kicsErrorCodes      = []string{"60", "50", "40", "30", "20"}
	containerResolver   wrappers.ContainerResolverWrapper
)

func NewScanCommand(
	applicationsWrapper wrappers.ApplicationsWrapper,
	scansWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	logsWrapper wrappers.LogsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	riskOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	jwtWrapper wrappers.JWTWrapper,
	scaRealTimeWrapper wrappers.ScaRealTimeWrapper,
	policyWrapper wrappers.PolicyWrapper,
	sastMetadataWrapper wrappers.SastMetadataWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	containerResolverWrapper wrappers.ContainerResolverWrapper,
	realtimeScannerWrapper wrappers.RealtimeScannerWrapper,
) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage scans",
		Long:  "The scan command enables the ability to manage scans in Checkmarx One",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html
			`,
			),
		},
	}

	createScanCmd := scanCreateSubCommand(
		scansWrapper,
		exportWrapper,
		resultsPdfReportsWrapper,
		resultsJSONReportsWrapper,
		uploadsWrapper,
		resultsWrapper,
		projectsWrapper,
		groupsWrapper,
		riskOverviewWrapper,
		scsScanOverviewWrapper,
		jwtWrapper,
		policyWrapper,
		accessManagementWrapper,
		applicationsWrapper,
		featureFlagsWrapper,
	)
	containerResolver = containerResolverWrapper

	listScansCmd := scanListSubCommand(scansWrapper, sastMetadataWrapper)

	showScanCmd := scanShowSubCommand(scansWrapper)

	scanASCACmd := scanASCASubCommand(jwtWrapper, featureFlagsWrapper)

	workflowScanCmd := scanWorkflowSubCommand(scansWrapper)

	deleteScanCmd := scanDeleteSubCommand(scansWrapper)

	cancelScanCmd := scanCancelSubCommand(scansWrapper)

	tagsCmd := scanTagsSubCommand(scansWrapper)

	logsCmd := scanLogsSubCommand(logsWrapper)

	kicsRealtimeCmd := scanRealtimeSubCommand()

	scaRealtimeCmd := scarealtime.NewScaRealtimeCommand(scaRealTimeWrapper)

	ossRealtimeCmd := scanOssRealtimeSubCommand(realtimeScannerWrapper, jwtWrapper, featureFlagsWrapper)

	containersRealtimeCmd := scanContainersRealtimeSubCommand(realtimeScannerWrapper, jwtWrapper, featureFlagsWrapper)

	secretsRealtimeCmd := scanSecretsRealtimeSubCommand(jwtWrapper, featureFlagsWrapper)

	iacRealtimeCmd := scanIacRealtimeSubCommand(jwtWrapper, featureFlagsWrapper)

	addFormatFlagToMultipleCommands(
		[]*cobra.Command{listScansCmd, showScanCmd, workflowScanCmd},
		printer.FormatTable, printer.FormatList, printer.FormatJSON,
	)
	addScanInfoFormatFlag(
		createScanCmd, printer.FormatList, printer.FormatTable, printer.FormatJSON,
	)
	scanCmd.AddCommand(
		createScanCmd,
		scanASCACmd,
		showScanCmd,
		workflowScanCmd,
		listScansCmd,
		deleteScanCmd,
		cancelScanCmd,
		tagsCmd,
		logsCmd,
		kicsRealtimeCmd,
		scaRealtimeCmd,
		ossRealtimeCmd,
		containersRealtimeCmd,
		secretsRealtimeCmd,
		iacRealtimeCmd,
	)
	return scanCmd
}

func scanRealtimeSubCommand() *cobra.Command {
	kicsContainerID := uuid.New()
	viper.Set(commonParams.KicsContainerNameKey, kicsContainerPrefixName+kicsContainerID.String())
	realtimeScanCmd := &cobra.Command{
		Use:   "kics-realtime",
		Short: "Create and run kics scan",
		Long:  "The kics-realtime command enables the ability to create, run and retrieve results from a kics scan using a docker image",
		Example: heredoc.Doc(
			`
			$ cx scan kics-realtime --file <file> --additional-params <additional-params> --engine <engine>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
			https://checkmarx.com/resource/documents/en/34965-68643-scan.html#UUID-350af120-85fa-9f20-7051-6d605524b4fc
			`,
			),
		},
		RunE: runKicksRealtime(),
	}
	realtimeScanCmd.PersistentFlags().
		StringSliceVar(
			&aditionalParameters, commonParams.KicsRealtimeAdditionalParams, []string{},
			"Additional scan options supported by kics. "+
				"Should follow comma separated format. For example : --additional-params -v, --exclude-results,fec62a97d569662093dbb9739360942f",
		)
	realtimeScanCmd.PersistentFlags().String(
		commonParams.KicsRealtimeFile,
		"",
		"Path to input file for kics realtime scanner",
	)
	realtimeScanCmd.PersistentFlags().String(
		commonParams.KicsRealtimeEngine,
		"docker",
		"Name in the $PATH for the container engine to run kics. Example:podman.",
	)
	markFlagAsRequired(realtimeScanCmd, commonParams.KicsRealtimeFile)
	return realtimeScanCmd
}

func scanLogsSubCommand(logsWrapper wrappers.LogsWrapper) *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Download scan log for selected scan type",
		Long:  "Accepts a scan-id and scan type (sast, iac-security) and downloads the related scan log",
		Example: heredoc.Doc(
			`
			$ cx scan logs --scan-id <scan Id> --scan-type <sast | iac-security>
		`,
		),
		RunE: runDownloadLogs(logsWrapper),
	}
	logsCmd.PersistentFlags().String(commonParams.ScanIDFlag, "", "Scan ID to retrieve log for.")
	logsCmd.PersistentFlags().String(commonParams.ScanTypeFlag, "", "Scan type to pull log for, ex: sast, iac-security")
	markFlagAsRequired(logsCmd, commonParams.ScanIDFlag)
	markFlagAsRequired(logsCmd, commonParams.ScanTypeFlag)

	return logsCmd
}

func scanTagsSubCommand(scansWrapper wrappers.ScansWrapper) *cobra.Command {
	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "Get a list of all available tags to filter by",
		Long:  "The tags command enables the ability to provide a list of all the available tags in Checkmarx One",
		Example: heredoc.Doc(
			`
			$ cx scan tags
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html#UUID-d1d53a56-197a-6a16-95e5-c437e6dc060a
			`,
			),
		},
		RunE: runGetTagsCommand(scansWrapper),
	}
	return tagsCmd
}

func scanCancelSubCommand(scansWrapper wrappers.ScansWrapper) *cobra.Command {
	cancelScanCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel one or more scans from running",
		Long:  "The cancel command enables the ability to cancel one or more running scans in Checkmarx One",
		Example: heredoc.Doc(
			`
			$ cx scan cancel --scan-id <scan ID>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html#UUID-800f2022-3609-3f40-6f77-9371e54f8b71
			`,
			),
		},
		RunE: runCancelScanCommand(scansWrapper),
	}
	addScanIDFlag(cancelScanCmd, "One or more scan IDs to cancel, ex: <scan-id>,<scan-id>,...")
	return cancelScanCmd
}

func scanDeleteSubCommand(scansWrapper wrappers.ScansWrapper) *cobra.Command {
	deleteScanCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes one or more scans",
		Example: heredoc.Doc(
			`
			$ cx scan delete --scan-id <scan Id>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html#UUID-851aa940-0454-ec17-4d29-42a2fa1352e0
			`,
			),
		},
		RunE: runDeleteScanCommand(scansWrapper),
	}
	addScanIDFlag(deleteScanCmd, "One or more scan IDs to delete, ex: <scan-id>,<scan-id>,...")
	return deleteScanCmd
}

func scanWorkflowSubCommand(scansWrapper wrappers.ScansWrapper) *cobra.Command {
	workflowScanCmd := &cobra.Command{
		Use:   "workflow <scan id>",
		Short: "Show information about a scan workflow",
		Long:  "The workflow command enables the ability to provide information about a requested scan workflow in Checkmarx One",
		Example: heredoc.Doc(
			`
			$ cx scan workflow --scan-id <scan Id>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html#UUID-9a524fb7-0dba-314d-9068-ccea184bc8d9
			`,
			),
		},
		RunE: runScanWorkflowByIDCommand(scansWrapper),
	}
	addScanIDFlag(workflowScanCmd, "Scan ID to workflow")
	return workflowScanCmd
}

func scanShowSubCommand(scansWrapper wrappers.ScansWrapper) *cobra.Command {
	showScanCmd := &cobra.Command{
		Use:   "show",
		Short: "Show information about a scan",
		Long:  "The show command enables the ability to show information about a requested scan in Checkmarx One",
		Example: heredoc.Doc(
			`
			$ cx scan show --scan-id <scan Id>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html#UUID-c073d85b-7605-0c89-909c-7d5b9caaec16
			`,
			),
		},
		RunE: runGetScanByIDCommand(scansWrapper),
	}
	addScanIDFlag(showScanCmd, "Scan ID to show")
	return showScanCmd
}

func scanASCASubCommand(jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) *cobra.Command {
	scanASCACmd := &cobra.Command{
		Hidden: true,
		Use:    "asca",
		Short:  "Run a ASCA scan",
		Long:   "Running a ASCA scan is a fast and efficient way to identify vulnerabilities in a specific file",
		Example: heredoc.Doc(
			`
			$ cx scan asca --file-source <path to a single file> --asca-latest-version
			$ cx scan asca --file-source <path to a single file> --ignored-file-path <path to ignored in file>	
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://docs.checkmarx.com/en/34965-68625-checkmarx-one-cli-commands.html
			`,
			),
		},
		RunE: asca.RunScanASCACommand(jwtWrapper),
	}

	scanASCACmd.PersistentFlags().Bool(commonParams.ASCALatestVersion, false,
		"Use this flag to update to the latest version of the ASCA scanner."+
			"Otherwise, we will check if there is an existing installation that can be used.")
	scanASCACmd.PersistentFlags().StringP(
		commonParams.SourcesFlag,
		commonParams.SourcesFlagSh,
		"",
		"The file source should be the path to a single file",
	)

	scanASCACmd.PersistentFlags().String(commonParams.IgnoredFilePathFlag, "", "Path to ignored secrets file")
	return scanASCACmd
}

func scanOssRealtimeSubCommand(
	realtimeScannerWrapper wrappers.RealtimeScannerWrapper,
	jwtWrapper wrappers.JWTWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) *cobra.Command {
	scanOssRealtimeCmd := &cobra.Command{
		Hidden: true,
		Use:    "oss-realtime",
		Short:  "Run a OSS-Realtime scan",
		Long:   "Running a OSS-Realtime scan is a fast and efficient way to identify malicious packages in a manifest file",
		Example: heredoc.Doc(
			`
			$ cx scan oss-realtime -s <path to a manifest file> --ignored-file-path <path to ignored packages JSON>
			`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://docs.checkmarx.com/en/34965-68625-checkmarx-one-cli-commands.html
				`,
			),
		},
		RunE: RunScanOssRealtimeCommand(realtimeScannerWrapper, jwtWrapper, featureFlagsWrapper),
	}

	scanOssRealtimeCmd.PersistentFlags().StringP(
		commonParams.SourcesFlag,
		commonParams.SourcesFlagSh,
		"",
		"The file source should be the path to a single file or multiple files separated by commas",
	)

	scanOssRealtimeCmd.Flags().String(
		commonParams.IgnoredFilePathFlag,
		"",
		"Path to a JSON file listing ignored packages",
	)

	return scanOssRealtimeCmd
}

func scanIacRealtimeSubCommand(
	jwtWrapper wrappers.JWTWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) *cobra.Command {
	scanIacRealtimeCmd := &cobra.Command{
		Hidden: true,
		Use:    "iac-realtime",
		Short:  "Run a IaC-Realtime scan",
		Long:   "Running a IaC-Realtime scan is a fast and efficient way to identify Infrustructure as Code vulnerabilities in a file.",
		Example: heredoc.Doc(
			`
			$ cx scan iac-realtime -s <path to a manifest file> --ignored-file-path <path to ignored iac vulnerabilities JSON file>
			`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://docs.checkmarx.com/en/34965-68625-checkmarx-one-cli-commands.html
				`,
			),
		},
		RunE: RunScanIacRealtimeCommand(jwtWrapper, featureFlagsWrapper),
	}

	scanIacRealtimeCmd.PersistentFlags().StringP(
		commonParams.SourcesFlag,
		commonParams.SourcesFlagSh,
		"",
		"The file source should be the path to a single file",
	)

	scanIacRealtimeCmd.Flags().String(
		commonParams.IgnoredFilePathFlag,
		"",
		"Path to a JSON file listing ignored iac vulnerabilities",
	)

	scanIacRealtimeCmd.Flags().String(
		commonParams.EngineFlag,
		"docker",
		"Name of the container engine to run IaC-Realtime. (ex. docker, podman)",
	)

	return scanIacRealtimeCmd
}

func scanContainersRealtimeSubCommand(realtimeScannerWrapper wrappers.RealtimeScannerWrapper, jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) *cobra.Command {
	scanContainersRealtimeCmd := &cobra.Command{
		Hidden: true,
		Use:    "containers-realtime",
		Short:  "Run a Containers-Realtime scan",
		Long:   "Running a Containers-Realtime scan is a fast and efficient way to identify vulnerabilities in container images",
		Example: heredoc.Doc(
			`
			$ cx scan containers-realtime -s <path to containers file>
			$ cx scan containers-realtime -s <path to file> --ignored-file-path <path to ignored containers file>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://docs.checkmarx.com/en/34965-68625-checkmarx-one-cli-commands.html
			`,
			),
		},
		RunE: RunScanContainersRealtimeCommand(realtimeScannerWrapper, jwtWrapper, featureFlagsWrapper),
	}

	scanContainersRealtimeCmd.PersistentFlags().StringP(
		commonParams.SourcesFlag,
		commonParams.SourcesFlagSh,
		"",
		"The file source should be the path to a single containers file (Dockerfile, docker-compose.yml, or Helm template)",
	)
	scanContainersRealtimeCmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "Path to ignored containers file")

	return scanContainersRealtimeCmd
}

func scanSecretsRealtimeSubCommand(
	jwtWrapper wrappers.JWTWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) *cobra.Command {
	scanSecretsRealtimeCmd := &cobra.Command{
		Hidden: true,
		Use:    "secrets-realtime",
		Short:  "Run a Secrets-Realtime scan",
		Long:   "Running a Secrets-Realtime scan is a fast and efficient way to identify exposed secrets in a file.",
		Example: heredoc.Doc(`
			$ cx scan secrets-realtime -s <path to file>
			$ cx scan secrets-realtime -s <path to file> --ignored-file-path <path to ignored secrets file>
		`),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(`
				https://docs.checkmarx.com/en/34965-68625-checkmarx-one-cli-commands.html
			`),
		},
		RunE: RunScanSecretsRealtimeCommand(jwtWrapper, featureFlagsWrapper),
	}

	scanSecretsRealtimeCmd.Flags().StringP(commonParams.SourcesFlag, "s", "", "Path to the file to scan")
	scanSecretsRealtimeCmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "Path to ignored secrets file")

	return scanSecretsRealtimeCmd
}

func scanListSubCommand(scansWrapper wrappers.ScansWrapper, sastMetadataWrapper wrappers.SastMetadataWrapper) *cobra.Command {
	listScansCmd := &cobra.Command{
		Use:   "list",
		Short: "List all scans in Checkmarx One",
		Long:  "The list command provides a list of all the scans in Checkmarx One",
		Example: heredoc.Doc(
			`
			$ cx scan list
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html#UUID-f92335a6-5b1c-e158-7914-2a4e72a2ada5
			`,
			),
		},
		RunE: runListScansCommand(scansWrapper, sastMetadataWrapper),
	}
	listScansCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterScanListFlagUsage)
	return listScansCmd
}

func scanCreateSubCommand(
	scansWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	jwtWrapper wrappers.JWTWrapper,
	policyWrapper wrappers.PolicyWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) *cobra.Command {
	createScanCmd := &cobra.Command{
		Use:   "create",
		Short: "Create and run a new scan",
		Long:  "The create command enables the ability to create and run a new scan in Checkmarx One",
		Example: heredoc.Doc(
			`
			$ cx scan create --project-name <Project Name> -s <path or repository url>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html#UUID-a0bb20d5-5182-3fb4-3da0-0e263344ffe7
			`,
			),
		},
		RunE: runCreateScanCommand(
			scansWrapper,
			exportWrapper,
			resultsPdfReportsWrapper,
			resultsJSONReportsWrapper,
			uploadsWrapper,
			resultsWrapper,
			projectsWrapper,
			groupsWrapper,
			risksOverviewWrapper,
			scsScanOverviewWrapper,
			jwtWrapper,
			policyWrapper,
			accessManagementWrapper,
			applicationsWrapper,
			featureFlagsWrapper,
		),
	}
	createScanCmd.PersistentFlags().Bool(commonParams.AsyncFlag, false, "Do not wait for scan completion")
	createScanCmd.PersistentFlags().IntP(
		commonParams.WaitDelayFlag,
		"",
		commonParams.WaitDelayDefault,
		"Polling wait time in seconds",
	)
	createScanCmd.PersistentFlags().Int(
		commonParams.ScanTimeoutFlag,
		0,
		"Cancel the scan and fail after the timeout in minutes",
	)
	createScanCmd.PersistentFlags().StringP(
		commonParams.SourcesFlag,
		commonParams.SourcesFlagSh,
		"",
		"Sources like: directory, zip file or git URL",
	)
	createScanCmd.PersistentFlags().StringP(
		commonParams.SourceDirFilterFlag,
		commonParams.SourceDirFilterFlagSh,
		"",
		"Source file filtering pattern",
	)
	createScanCmd.PersistentFlags().StringP(
		commonParams.IncludeFilterFlag,
		commonParams.IncludeFilterFlagSh,
		"",
		"Only files scannable by AST are included by default."+
			" Add a comma separated list of extra inclusions, ex: *zip,file.txt",
	)
	createScanCmd.PersistentFlags().String(commonParams.ProjectName, "", "Name of the project")
	err := createScanCmd.MarkPersistentFlagRequired(commonParams.ProjectName)
	if err != nil {
		log.Fatal(err)
	}
	createScanCmd.PersistentFlags().Bool(
		commonParams.IncrementalSast,
		false,
		"Incremental SAST scan should be performed.",
	)

	createScanCmd.PersistentFlags().String(commonParams.PresetName, "", "The name of the Checkmarx preset to use.")
	createScanCmd.PersistentFlags().String(commonParams.IacsPresetIDFlag, "", commonParams.IacsPresetIDUsage)

	createScanCmd.PersistentFlags().String(
		commonParams.ScaResolverFlag,
		"",
		"Resolve SCA project dependencies (path to SCA Resolver executable)",
	)
	createScanCmd.PersistentFlags().String(
		commonParams.ScaResolverParamsFlag,
		"",
		fmt.Sprintf("Parameters to use in SCA resolver (requires --%s)", commonParams.ScaResolverFlag),
	)
	createScanCmd.PersistentFlags().Bool(commonParams.ContainerResolveLocallyFlag, false, "Execute container resolver locally.")
	createScanCmd.PersistentFlags().String(commonParams.ContainerImagesFlag, "", "List of container images to scan, ex: manuelbcd/vulnapp:latest,debian:10")
	createScanCmd.PersistentFlags().String(commonParams.ScanTypes, "", "Scan types, ex: (sast,iac-security,sca,api-security)")

	createScanCmd.PersistentFlags().String(commonParams.TagList, "", "List of tags, ex: (tagA,tagB:val,etc)")
	createScanCmd.PersistentFlags().StringP(
		commonParams.BranchFlag, commonParams.BranchFlagSh,
		commonParams.Branch, commonParams.BranchFlagUsage,
	)
	createScanCmd.PersistentFlags().String(commonParams.SastFilterFlag, "", commonParams.SastFilterUsage)
	createScanCmd.PersistentFlags().String(commonParams.IacsFilterFlag, "", commonParams.IacsFilterUsage)
	createScanCmd.PersistentFlags().String(commonParams.KicsFilterFlag, "", commonParams.KicsFilterUsage)

	err = createScanCmd.PersistentFlags().MarkDeprecated(commonParams.KicsFilterFlag, "please use the replacement flag --iac-security-filter")
	if err != nil {
		return nil
	}

	createScanCmd.PersistentFlags().StringSlice(
		commonParams.KicsPlatformsFlag,
		[]string{},
		commonParams.KicsPlatformsFlagUsage,
	)
	createScanCmd.PersistentFlags().Bool(
		commonParams.SastFastScanFlag,
		false,
		"Enable SAST Fast Scan configuration",
	)

	createScanCmd.PersistentFlags().Bool(
		commonParams.SastLightQueriesFlag,
		false,
		"Enable SAST scan using light query configuration",
	)

	createScanCmd.PersistentFlags().Bool(
		commonParams.BranchPrimaryFlag,
		false,
		"This flag sets the branch specified in --branch as the PRIMARY branch for the project")

	createScanCmd.PersistentFlags().Bool(
		commonParams.SastRecommendedExclusionsFlags,
		false,
		"Enable recommended exclusions configuration for SAST scan",
	)

	createScanCmd.PersistentFlags().StringSlice(
		commonParams.IacsPlatformsFlag,
		[]string{},
		commonParams.IacsPlatformsFlagUsage,
	)
	err = createScanCmd.PersistentFlags().MarkDeprecated(commonParams.KicsPlatformsFlag, "please use the replacement flag --iac-security-platforms")
	if err != nil {
		return nil
	}
	createScanCmd.PersistentFlags().String(commonParams.ScaFilterFlag, "", commonParams.ScaFilterUsage)
	createScanCmd.PersistentFlags().Bool(commonParams.SastRedundancyFlag, false, fmt.Sprintf(
		"Populate SAST results 'data.redundancy' with values '%s' (to fix) or '%s' (no need to fix)", fixLabel, redundantLabel))

	addResultFormatFlag(
		createScanCmd,
		printer.FormatSummaryConsole,
		printer.FormatJSON,
		printer.FormatJSONv2,
		printer.FormatSummary,
		printer.FormatSarif,
		printer.FormatSbom,
		printer.FormatPDF,
		printer.FormatSummaryMarkdown,
		printer.FormatGLSast,
		printer.FormatGLSca,
	)
	createScanCmd.PersistentFlags().String(commonParams.APIDocumentationFlag, "", apiDocumentationFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ExploitablePathFlag, "", exploitablePathFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.LastSastScanTime, "", scaLastScanTimeFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ProjecPrivatePackageFlag, "", projectPrivatePackageFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ScaPrivatePackageVersionFlag, "", scaPrivatePackageVersionFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ReportFormatPdfToEmailFlag, "", pdfToEmailFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ReportSbomFormatFlag, services.DefaultSbomOption, sbomReportFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ReportFormatPdfOptionsFlag, defaultPdfOptionsDataSections, pdfOptionsFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.TargetFlag, "cx_result", "Output file")
	createScanCmd.PersistentFlags().String(commonParams.TargetPathFlag, ".", "Output Path")
	createScanCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterResultsListFlagUsage)
	createScanCmd.PersistentFlags().String(commonParams.ProjectGroupList, "", "List of groups to associate to project")
	createScanCmd.PersistentFlags().String(commonParams.ProjectTagList, "", "List of tags to associate to project")
	createScanCmd.PersistentFlags().String(
		commonParams.Threshold,
		"",
		commonParams.ThresholdFlagUsage,
	)
	createScanCmd.PersistentFlags().Bool(
		commonParams.ScanResubmit,
		false,
		"Create a scan with the configurations used in the most recent scan in the project",
	)
	createScanCmd.PersistentFlags().Int(
		commonParams.PolicyTimeoutFlag,
		commonParams.ScanPolicyDefaultTimeout,
		"Cancel the policy evaluation and fail after the timeout in minutes",
	)
	createScanCmd.PersistentFlags().Bool(commonParams.IgnorePolicyFlag, false, "Do not evaluate policies")
	_ = createScanCmd.PersistentFlags().MarkHidden(commonParams.IgnorePolicyFlag)

	createScanCmd.PersistentFlags().String(commonParams.ApplicationName, "", "Name of the application to assign with the project")
	// Link the environment variables to the CLI argument(s).
	err = viper.BindPFlag(commonParams.BranchKey, createScanCmd.PersistentFlags().Lookup(commonParams.BranchFlag))
	if err != nil {
		log.Fatal(err)
	}

	createScanCmd.PersistentFlags().String(commonParams.SSHKeyFlag, "", "Path to ssh private key")

	createScanCmd.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "Provide a token with read permission for the repo that you are scanning (for scorecard scans)")
	createScanCmd.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "The URL of the repo that you are scanning with scs (for scorecard scans)")
	createScanCmd.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "Specify which scs engines will run (default: all licensed engines)")
	createScanCmd.PersistentFlags().Bool(commonParams.ScaHideDevAndTestDepFlag, false, scaHideDevAndTestDepFlagDescription)

	// Container config flags
	createScanCmd.PersistentFlags().String(commonParams.ContainersFileFolderFilterFlag, "", "Specify files and folders to be included or excluded from scans, ex: \"!*.log\"")
	createScanCmd.PersistentFlags().String(commonParams.ContainersPackageFilterFlag, "", "Exclude packages by package name or file path using regex, ex: \"^internal-.*\"")
	createScanCmd.PersistentFlags().Bool(commonParams.ContainersExcludeNonFinalStagesFlag, false, "Scan only the final deployable image")
	createScanCmd.PersistentFlags().String(commonParams.ContainersImageTagFilterFlag, "", "Exclude images by image name and/or tag, ex: \"*dev\"")

	// reading sbom-only flag
	createScanCmd.PersistentFlags().Bool(commonParams.SbomFlag, false, "Scan only the specified SBOM file (supported formats xml or json)")

	return createScanCmd
}

func setupScanTags(input *[]byte, cmd *cobra.Command) {
	tagListStr, _ := cmd.Flags().GetString(commonParams.TagList)
	tags := strings.Split(tagListStr, ",")
	var info map[string]interface{}
	_ = json.Unmarshal(*input, &info)
	if _, ok := info["tags"]; !ok {
		var tagMap map[string]interface{}
		_ = json.Unmarshal([]byte("{}"), &tagMap)
		info["tags"] = tagMap
	}
	for _, tag := range tags {
		if tag != "" {
			value := ""
			keyValuePair := strings.Split(tag, ":")
			if len(keyValuePair) > 1 {
				value = keyValuePair[1]
			}
			info["tags"].(map[string]interface{})[keyValuePair[0]] = value
		}
	}
	*input, _ = json.Marshal(info)
}

func setupScanTypeProjectAndConfig(
	input *[]byte,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	scansWrapper wrappers.ScansWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	jwtWrapper wrappers.JWTWrapper,
) error {
	userAllowedEngines, _ := jwtWrapper.GetAllowedEngines(featureFlagsWrapper)
	var info map[string]interface{}
	newProjectName, _ := cmd.Flags().GetString(commonParams.ProjectName)

	_ = json.Unmarshal(*input, &info)
	info[resultsMapType] = getUploadType(cmd)
	// Handle the project settings
	if _, ok := info["project"]; !ok {
		var projectMap map[string]interface{}
		_ = json.Unmarshal([]byte("{}"), &projectMap)
		info["project"] = projectMap
	}
	if newProjectName != "" {
		info["project"].(map[string]interface{})["id"] = newProjectName
	} else {
		return errors.New("Project name is required")
	}

	// We need to convert the project name into an ID
	projectID, findProjectErr := services.FindProject(
		info["project"].(map[string]interface{})["id"].(string),
		cmd,
		projectsWrapper,
		groupsWrapper,
		accessManagementWrapper,
		applicationsWrapper,
		featureFlagsWrapper,
	)
	if findProjectErr != nil {
		return findProjectErr
	}

	info["project"].(map[string]interface{})["id"] = projectID
	// Handle the scan configuration
	var configArr []interface{}
	resubmit, _ := cmd.Flags().GetBool(commonParams.ScanResubmit)
	var resubmitConfig []wrappers.Config
	if resubmit {
		logger.PrintIfVerbose(
			fmt.Sprintf(
				"using latest scan configuration due to --%s flag",
				commonParams.ScanResubmit,
			),
		)
		userScanTypes, _ := cmd.Flags().GetString(commonParams.ScanTypes)
		// Get the latest scan configuration
		resubmitConfig, _ = getResubmitConfiguration(scansWrapper, projectID, userScanTypes)
	} else if _, ok := info["config"]; !ok {
		err := json.Unmarshal([]byte("[]"), &configArr)
		if err != nil {
			return err
		}
	}

	sastConfig := addSastScan(cmd, resubmitConfig)
	if sastConfig != nil {
		configArr = append(configArr, sastConfig)
	}
	var kicsConfig = addKicsScan(cmd, resubmitConfig)
	if kicsConfig != nil {
		configArr = append(configArr, kicsConfig)
	}
	var scaConfig = addScaScan(cmd, resubmitConfig, userAllowedEngines[commonParams.ContainersType])
	if scaConfig != nil {
		configArr = append(configArr, scaConfig)
	}
	var apiSecConfig = addAPISecScan(cmd)
	if apiSecConfig != nil {
		configArr = append(configArr, apiSecConfig)
	}
	var containersConfig, containersErr = addContainersScan(cmd, resubmitConfig)
	if containersErr != nil {
		return containersErr
	} else if containersConfig != nil {
		configArr = append(configArr, containersConfig)
	}

	var SCSConfig, scsErr = addSCSScan(cmd, resubmitConfig, userAllowedEngines[commonParams.EnterpriseSecretsType])
	if scsErr != nil {
		return scsErr
	} else if SCSConfig != nil {
		configArr = append(configArr, SCSConfig)
	}

	info["config"] = configArr
	var err2 error
	*input, err2 = json.Marshal(info)
	if err2 != nil {
		return err2
	}

	return nil
}

func getResubmitConfiguration(scansWrapper wrappers.ScansWrapper, projectID, userScanTypes string) (
	[]wrappers.Config,
	error,
) {
	var allScansModel *wrappers.ScansCollectionResponseModel
	var errorModel *wrappers.ErrorModel
	var err error
	var config []wrappers.Config
	params := make(map[string]string)
	params["project-id"] = projectID
	allScansModel, errorModel, err = scansWrapper.Get(params)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s\n", failedGettingAll)
	}
	// Checking the response for errors
	if errorModel != nil {
		return nil, errors.Errorf(services.ErrorCodeFormat, failedGettingAll, errorModel.Code, errorModel.Message)
	}

	if len(allScansModel.Scans) > 0 {
		scanModelResponse := allScansModel.Scans[0]
		config = scanModelResponse.Metadata.Configs
		engines := scanModelResponse.Engines
		// Check if there are no scan types sent using the flags, and use the latest scan engine types
		if userScanTypes == "" {
			actualScanTypes = strings.Join(engines, ",")
		}
	}

	return config, nil
}

func addSastScan(cmd *cobra.Command, resubmitConfig []wrappers.Config) map[string]interface{} {
	// Check if SAST is enabled
	if !scanTypeEnabled(commonParams.SastType) {
		return nil
	}

	sastMapConfig := make(map[string]interface{})
	sastConfig := wrappers.SastConfig{}
	sastMapConfig[resultsMapType] = commonParams.SastType

	sastFastScanChanged := cmd.Flags().Changed(commonParams.SastFastScanFlag)
	sastLightQueryChanged := cmd.Flags().Changed(commonParams.SastLightQueriesFlag)
	sastRecommendedExclusionsChanged := cmd.Flags().Changed(commonParams.SastRecommendedExclusionsFlags)

	sastIncrementalChanged := cmd.Flags().Changed(commonParams.IncrementalSast)

	if sastFastScanChanged {
		fastScan, _ := cmd.Flags().GetBool(commonParams.SastFastScanFlag)
		sastConfig.FastScanMode = strconv.FormatBool(fastScan)
	}

	if sastLightQueryChanged {
		lightQuery, _ := cmd.Flags().GetBool(commonParams.SastLightQueriesFlag)
		sastConfig.LightQueries = strconv.FormatBool(lightQuery)
	}

	if sastRecommendedExclusionsChanged {
		recommendedExclusions, _ := cmd.Flags().GetBool(commonParams.SastRecommendedExclusionsFlags)
		sastConfig.RecommendedExclusions = strconv.FormatBool(recommendedExclusions)
	}

	if sastIncrementalChanged {
		incrementalVal, _ := cmd.Flags().GetBool(commonParams.IncrementalSast)
		sastConfig.Incremental = strconv.FormatBool(incrementalVal)
	}

	sastConfig.PresetName, _ = cmd.Flags().GetString(commonParams.PresetName)
	sastConfig.Filter, _ = cmd.Flags().GetString(commonParams.SastFilterFlag)

	for _, config := range resubmitConfig {
		if config.Type != commonParams.SastType {
			continue
		}

		overrideSastConfigValue(sastFastScanChanged, sastIncrementalChanged, sastLightQueryChanged, sastRecommendedExclusionsChanged, &sastConfig, config)
	}

	sastMapConfig[resultsMapValue] = &sastConfig
	return sastMapConfig
}

func overrideSastConfigValue(sastFastScanChanged, sastIncrementalChanged, sastLightQueryChanged, sastRecommendedExclusionsChanged bool, sastConfig *wrappers.SastConfig, config wrappers.Config) {
	setIfEmpty := func(configValue *string, resubmitValue interface{}) {
		if *configValue == "" && resubmitValue != nil {
			*configValue = resubmitValue.(string)
		}
	}

	if resubmitIncremental := config.Value[configIncremental]; resubmitIncremental != nil && !sastIncrementalChanged {
		sastConfig.Incremental = resubmitIncremental.(string)
	}
	if resubmitFastScan := config.Value[configFastScan]; resubmitFastScan != nil && !sastFastScanChanged {
		sastConfig.FastScanMode = resubmitFastScan.(string)
	}

	if resubmitLightQuery := config.Value[configLightQueries]; resubmitLightQuery != nil && !sastLightQueryChanged {
		sastConfig.LightQueries = resubmitLightQuery.(string)
	}

	if resubmitRecommendedExclusions := config.Value[configRecommendedExclusions]; resubmitRecommendedExclusions != nil && !sastRecommendedExclusionsChanged {
		sastConfig.RecommendedExclusions = resubmitRecommendedExclusions.(string)
	}

	setIfEmpty(&sastConfig.PresetName, config.Value[configPresetName])
	setIfEmpty(&sastConfig.Filter, config.Value[configFilterKey])

	if resubmitEngineVerbose := config.Value[configEngineVerbose]; resubmitEngineVerbose != nil {
		sastConfig.EngineVerbose = resubmitEngineVerbose.(string)
	}
	if resubmitLanguageMode := config.Value[configLanguageMode]; resubmitLanguageMode != nil {
		sastConfig.LanguageMode = resubmitLanguageMode.(string)
	}
}

func addKicsScan(cmd *cobra.Command, resubmitConfig []wrappers.Config) map[string]interface{} {
	if scanTypeEnabled(commonParams.KicsType) {
		kicsMapConfig := make(map[string]interface{})
		kicsConfig := wrappers.KicsConfig{}
		kicsMapConfig[resultsMapType] = commonParams.KicsType
		kicsConfig.Filter = deprecatedFlagValue(cmd, commonParams.KicsFilterFlag, commonParams.IacsFilterFlag)
		kicsConfig.Platforms = deprecatedFlagValue(cmd, commonParams.KicsPlatformsFlag, commonParams.IacsPlatformsFlag)
		kicsConfig.PresetID, _ = cmd.Flags().GetString(commonParams.IacsPresetIDFlag)
		for _, config := range resubmitConfig {
			if config.Type == commonParams.KicsType {
				resubmitFilter := config.Value[configFilterKey]
				if resubmitFilter != nil && kicsConfig.Filter == "" {
					kicsConfig.Filter = resubmitFilter.(string)
				}
				resubmitPlatforms := config.Value[configFilterPlatforms]
				if resubmitPlatforms != nil && kicsConfig.Platforms == "" {
					kicsConfig.Platforms = resubmitPlatforms.(string)
				}
				resubmitPresetID := config.Value[configPresetID]
				if resubmitPresetID != nil && kicsConfig.PresetID == "" {
					kicsConfig.PresetID = resubmitPresetID.(string)
				}
			}
		}
		kicsMapConfig[resultsMapValue] = &kicsConfig
		return kicsMapConfig
	}
	return nil
}

func addScaScan(cmd *cobra.Command, resubmitConfig []wrappers.Config, hasContainerLicense bool) map[string]interface{} {
	if scanTypeEnabled(commonParams.ScaType) {
		scaMapConfig := make(map[string]interface{})
		scaConfig := wrappers.ScaConfig{}
		scaMapConfig[resultsMapType] = commonParams.ScaType
		isSbom, _ := cmd.PersistentFlags().GetBool(commonParams.SbomFlag)
		scaConfig.Filter, _ = cmd.Flags().GetString(commonParams.ScaFilterFlag)
		scaConfig.LastSastScanTime, _ = cmd.Flags().GetString(commonParams.LastSastScanTime)
		scaConfig.PrivatePackageVersion, _ = cmd.Flags().GetString(commonParams.ScaPrivatePackageVersionFlag)
		if hasContainerLicense {
			scaConfig.EnableContainersScan = false
		}
		exploitablePath, _ := cmd.Flags().GetString(commonParams.ExploitablePathFlag)
		if exploitablePath != "" {
			scaConfig.ExploitablePath = strings.ToLower(exploitablePath)
		}
		for _, config := range resubmitConfig {
			if config.Type == commonParams.ScaType {
				resubmitFilter := config.Value[configFilterKey]
				if resubmitFilter != nil && scaConfig.Filter == "" {
					scaConfig.Filter = resubmitFilter.(string)
				}
			}
		}
		scaConfig.SBom = strconv.FormatBool(isSbom)
		scaMapConfig[resultsMapValue] = &scaConfig
		return scaMapConfig
	}
	return nil
}

func addContainersScan(cmd *cobra.Command, resubmitConfig []wrappers.Config) (map[string]interface{}, error) {
	if !scanTypeEnabled(commonParams.ContainersType) {
		return nil, nil
	}
	containerMapConfig := make(map[string]interface{})
	containerMapConfig[resultsMapType] = commonParams.ContainersType
	containerConfig := wrappers.ContainerConfig{}

	containerResolveLocally, _ := cmd.Flags().GetBool(commonParams.ContainerResolveLocallyFlag)
	initializeContainersConfigWithResubmitValues(resubmitConfig, &containerConfig, containerResolveLocally)

	fileFolderFilter, _ := cmd.PersistentFlags().GetString(commonParams.ContainersFileFolderFilterFlag)
	if fileFolderFilter != "" {
		containerConfig.FilesFilter = fileFolderFilter
	}
	packageFilter, _ := cmd.PersistentFlags().GetString(commonParams.ContainersPackageFilterFlag)
	if packageFilter != "" {
		containerConfig.PackagesFilter = packageFilter
	}
	excludeNonFinalStages, _ := cmd.PersistentFlags().GetBool(commonParams.ContainersExcludeNonFinalStagesFlag)
	if cmd.PersistentFlags().Changed(commonParams.ContainersExcludeNonFinalStagesFlag) {
		containerConfig.NonFinalStagesFilter = strconv.FormatBool(excludeNonFinalStages)
	}
	imageTagFilter, _ := cmd.Flags().GetString(commonParams.ContainersImageTagFilterFlag)
	if imageTagFilter != "" {
		containerConfig.ImagesFilter = imageTagFilter
	}
	userCustomImages, _ := cmd.Flags().GetString(commonParams.ContainerImagesFlag)
	if userCustomImages != "" && !containerResolveLocally {
		containerImagesList := strings.Split(strings.TrimSpace(userCustomImages), ",")
		for _, containerImageName := range containerImagesList {
			if containerImagesErr := validateContainerImageFormat(containerImageName); containerImagesErr != nil {
				return nil, containerImagesErr
			}
		}
		logger.PrintIfVerbose(fmt.Sprintf("User input container images identified: %v", strings.Join(containerImagesList, ", ")))
		containerConfig.UserCustomImages = userCustomImages
	}

	containerMapConfig[resultsMapValue] = &containerConfig
	return containerMapConfig, nil
}

func initializeContainersConfigWithResubmitValues(resubmitConfig []wrappers.Config, containerConfig *wrappers.ContainerConfig, containerResolveLocally bool) {
	for _, config := range resubmitConfig {
		if config.Type != commonParams.ContainersType {
			continue
		}
		resubmitFilesFilter := config.Value[ConfigContainersFilesFilterKey]
		if resubmitFilesFilter != nil && resubmitFilesFilter != "" {
			containerConfig.FilesFilter = resubmitFilesFilter.(string)
		}
		resubmitPackagesFilter := config.Value[ConfigContainersPackagesFilterKey]
		if resubmitPackagesFilter != nil && resubmitPackagesFilter != "" {
			containerConfig.PackagesFilter = resubmitPackagesFilter.(string)
		}
		resubmitNonFinalStagesFilter := config.Value[ConfigContainersNonFinalStagesFilterKey]
		if resubmitNonFinalStagesFilter != nil && resubmitNonFinalStagesFilter != "" {
			containerConfig.NonFinalStagesFilter = resubmitNonFinalStagesFilter.(string)
		}
		resubmitImagesFilter := config.Value[ConfigContainersImagesFilterKey]
		if resubmitImagesFilter != nil && resubmitImagesFilter != "" {
			containerConfig.ImagesFilter = resubmitImagesFilter.(string)
		}
		resubmitUserCustomImages := config.Value[ConfigUserCustomImagesKey]
		if resubmitUserCustomImages != nil && resubmitUserCustomImages != "" && !containerResolveLocally {
			containerConfig.UserCustomImages = resubmitUserCustomImages.(string)
		}
	}
}

func addAPISecScan(cmd *cobra.Command) map[string]interface{} {
	if scanTypeEnabled(commonParams.APISecurityType) {
		apiSecMapConfig := make(map[string]interface{})
		apiSecConfig := wrappers.APISecConfig{}
		apiSecMapConfig[resultsMapType] = commonParams.APISecType
		apiDocumentation, _ := cmd.Flags().GetString(commonParams.APIDocumentationFlag)
		if apiDocumentation != "" {
			apiSecConfig.SwaggerFilter = strings.ToLower(apiDocumentation)
		}
		apiSecMapConfig[resultsMapValue] = &apiSecConfig
		return apiSecMapConfig
	}
	return nil
}

func createResubmitConfig(resubmitConfig []wrappers.Config, scsRepoToken, scsRepoURL string, hasEnterpriseSecretsLicense bool) wrappers.SCSConfig {
	scsConfig := wrappers.SCSConfig{}
	for _, config := range resubmitConfig {
		resubmitTwoms := config.Value[configTwoms]
		if resubmitTwoms != nil && hasEnterpriseSecretsLicense {
			scsConfig.Twoms = resubmitTwoms.(string)
		}
		scsConfig.RepoURL = scsRepoURL
		scsConfig.RepoToken = scsRepoToken
		resubmitScoreCard := config.Value[ScsScoreCardType]
		if resubmitScoreCard == trueString && scsRepoToken != "" && scsRepoURL != "" {
			scsConfig.Scorecard = trueString
		} else {
			scsConfig.Scorecard = falseString
		}
	}
	return scsConfig
}

func getSCSEnginesSelected(scsEngines string) (isScorecardSelected, isSecretDetectionSelected bool) {
	if scsEngines == "" {
		return true, true
	}

	scsEnginesTypes := strings.Split(scsEngines, ",")
	for _, engineType := range scsEnginesTypes {
		engineType = strings.TrimSpace(engineType)
		switch engineType {
		case ScsSecretDetectionType:
			isSecretDetectionSelected = true
		case ScsScoreCardType:
			isScorecardSelected = true
		}
	}

	return isScorecardSelected, isSecretDetectionSelected
}

func isURLSupportedByScorecard(scsRepoURL string) bool {
	// only for https; currently our scorecard solution doesn't support GitHub Enterprise Server hosts
	githubURLPattern := regexp.MustCompile(`(?:https?://)?(?:^|[^.])github\.com/.+`)
	isGithubURL := githubURLPattern.MatchString(scsRepoURL)
	if scsRepoURL != "" && !isGithubURL {
		fmt.Println(ScsScorecardUnsupportedHostWarningMsg)
	}
	return isGithubURL
}

func isScorecardRunnable(isScsEnginesFlagSet, scsScorecardSelected bool, scsRepoToken, scsRepoURL, userScanTypes string) (bool, error) {
	if scsRepoToken == "" || scsRepoURL == "" {
		// with --scs-engine "scorecard" set, if flags not defined, scorecard will launch an error
		if userScanTypes != "" && isScsEnginesFlagSet && scsScorecardSelected {
			return false, errors.New(ScsRepoRequiredMsg)
		}

		// with no --scan-types flag or --scan-types "scs", if flags not defined, scorecard will launch a warning, otherwise will run
		fmt.Println(ScsRepoWarningMsg)
		return false, nil
	}

	return isURLSupportedByScorecard(scsRepoURL), nil
}

func addSCSScan(cmd *cobra.Command, resubmitConfig []wrappers.Config, hasEnterpriseSecretsLicense bool) (map[string]interface{}, error) {
	if !scanTypeEnabled(commonParams.ScsType) && !scanTypeEnabled(commonParams.MicroEnginesType) {
		return nil, nil
	}
	scsConfig := wrappers.SCSConfig{}
	scsMapConfig := make(map[string]interface{})

	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType // scs is still microengines in the scans API
	userScanTypes, _ := cmd.Flags().GetString(commonParams.ScanTypes)

	scsRepoToken := viper.GetString(commonParams.ScsRepoTokenKey)
	if token, _ := cmd.Flags().GetString(commonParams.SCSRepoTokenFlag); token != "" {
		scsRepoToken = token
	}

	viper.Set(commonParams.SCSRepoTokenFlag, scsRepoToken)
	scsRepoURL, _ := cmd.Flags().GetString(commonParams.SCSRepoURLFlag)

	viper.Set(commonParams.SCSRepoURLFlag, scsRepoURL)
	scsEngines, _ := cmd.Flags().GetString(commonParams.SCSEnginesFlag)

	if resubmitConfig != nil {
		scsConfig = createResubmitConfig(resubmitConfig, scsRepoToken, scsRepoURL, hasEnterpriseSecretsLicense)
		scsMapConfig[resultsMapValue] = &scsConfig
		return scsMapConfig, nil
	}

	scsScoreCardSelected, scsSecretDetectionSelected := getSCSEnginesSelected(scsEngines) // secret-detection or scorecard

	if scsSecretDetectionSelected && hasEnterpriseSecretsLicense {
		scsConfig.Twoms = trueString
	}

	isScsEnginesFlagSet := scsEngines != ""

	if scsScoreCardSelected {
		canRunScorecard, err := isScorecardRunnable(isScsEnginesFlagSet, scsScoreCardSelected, scsRepoToken, scsRepoURL, userScanTypes)
		if err != nil {
			return nil, err
		}

		if canRunScorecard {
			scsConfig.Scorecard = trueString
			scsConfig.RepoToken = scsRepoToken
			scsConfig.RepoURL = strings.ToLower(scsRepoURL)
		}
	}

	if scsConfig.Scorecard != trueString && scsConfig.Twoms != trueString {
		return nil, nil
	}

	scsMapConfig[resultsMapValue] = &scsConfig
	return scsMapConfig, nil
}

func validateScanTypes(cmd *cobra.Command, jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) error {
	var scanTypes []string
	var SCSScanTypes []string

	isSbomScan, _ := cmd.PersistentFlags().GetBool(commonParams.SbomFlag)

	allowedEngines, err := jwtWrapper.GetAllowedEngines(featureFlagsWrapper)
	if err != nil {
		err = errors.Errorf("Error validating scan types: %v", err)
		return err
	}

	userScanTypes, _ := cmd.Flags().GetString(commonParams.ScanTypes)
	userSCSScanTypes, _ := cmd.Flags().GetString(commonParams.SCSEnginesFlag)
	if len(userScanTypes) > 0 {
		userScanTypes = strings.ReplaceAll(strings.ToLower(userScanTypes), " ", "")
		userScanTypes = strings.Replace(strings.ToLower(userScanTypes), commonParams.KicsType, commonParams.IacType, 1)
		userScanTypes = strings.Replace(strings.ToLower(userScanTypes), commonParams.ContainersTypeFlag, commonParams.ContainersType, 1)
		userSCSScanTypes = strings.Replace(strings.ToLower(userSCSScanTypes), commonParams.SCSEnginesFlag, commonParams.ScsType, 1)

		scanTypes = strings.Split(userScanTypes, ",")

		// check scan-types, when sbom-only flag is used
		if isSbomScan {
			if len(scanTypes) > 1 {
				err = errors.Errorf(sbomScanTypeErrMsg)
				return err
			}

			if scanTypes[0] != "sca" {
				err = errors.Errorf(sbomScanTypeErrMsg)
				return err
			}
		}

		for _, scanType := range scanTypes {
			if !allowedEngines[scanType] {
				keys := reflect.ValueOf(allowedEngines).MapKeys()
				err = errors.Errorf(engineNotAllowed, scanType, scanType, keys)
				return err
			}
		}

		SCSScanTypes = strings.Split(userSCSScanTypes, ",")
		if slices.Contains(SCSScanTypes, ScsSecretDetectionType) && !allowedEngines[commonParams.EnterpriseSecretsType] {
			keys := reflect.ValueOf(allowedEngines).MapKeys()
			err = errors.Errorf(engineNotAllowed, ScsSecretDetectionType, ScsSecretDetectionType, keys)
			return err
		}
	} else {
		if isSbomScan {
			if allowedEngines["sca"] {
				// for sbom-flag, setting scan-type as only "sca"
				scanTypes = append(scanTypes, "sca")
			} else {
				return errors.Errorf("sbom needs sca engine to be allowed")
			}
		} else {
			for k := range allowedEngines {
				scanTypes = append(scanTypes, k)
			}
		}
	}
	actualScanTypes = strings.Join(scanTypes, ",")
	actualScanTypes = strings.Replace(strings.ToLower(actualScanTypes), commonParams.IacType, commonParams.KicsType, 1)

	return nil
}

func scanTypeEnabled(scanType string) bool {
	scanTypes := strings.Split(actualScanTypes, ",")
	for _, a := range scanTypes {
		if strings.EqualFold(strings.TrimSpace(a), scanType) {
			return true
		}
	}
	return false
}

func compressFolder(sourceDir, filter, userIncludeFilter, scaResolver string) (string, error) {
	scaToolPath := scaResolver
	outputFile, err := os.CreateTemp(os.TempDir(), "cx-*.zip")
	if err != nil {
		return "", errors.Wrapf(err, "Cannot source code temp file.")
	}
	defer outputFile.Close()
	zipWriter := zip.NewWriter(outputFile)
	err = addDirFiles(zipWriter, "", sourceDir, getExcludeFilters(filter), getIncludeFilters(userIncludeFilter))
	if err != nil {
		return "", err
	}
	if len(scaToolPath) > 0 && len(scaResolverResultsFile) > 0 {
		err = addScaResults(zipWriter)
		if err != nil {
			return "", err
		}
	}
	// Close the file
	err = zipWriter.Close()
	if err != nil {
		return "", err
	}
	stat, err := outputFile.Stat()
	if err != nil {
		return "", err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Zip size:  %.2fMB\n", float64(stat.Size())/mbBytes))
	return outputFile.Name(), err
}

func isSingleContainerScanTriggered() bool {
	scanTypeList := strings.Split(actualScanTypes, ",")
	return len(scanTypeList) == 1 && scanTypeList[0] == commonParams.ContainersType
}

func getIncludeFilters(userIncludeFilter string) []string {
	return buildFilters(commonParams.BaseIncludeFilters, userIncludeFilter)
}

func getExcludeFilters(userExcludeFilter string) []string {
	return buildFilters(commonParams.BaseExcludeFilters, userExcludeFilter)
}

func buildFilters(base []string, extra string) []string {
	if len(extra) > 0 {
		return append(base, strings.Split(extra, ",")...)
	}
	return base
}

func addDirFilesIgnoreFilter(zipWriter *zip.Writer, baseDir, parentDir string) error {
	files, err := ioutil.ReadDir(parentDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			logger.PrintIfVerbose("Directory: " + file.Name())
			newParent := parentDir + file.Name() + "/"
			newBase := baseDir + file.Name() + "/"
			err = addDirFilesIgnoreFilter(zipWriter, newBase, newParent)
		} else {
			fileName := parentDir + file.Name()
			logger.PrintIfVerbose("Included: " + fileName)
			dat, _ := ioutil.ReadFile(fileName)

			f, _ := zipWriter.Create(baseDir + file.Name())

			_, err = f.Write(dat)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func addDirFiles(zipWriter *zip.Writer, baseDir, parentDir string, filters, includeFilters []string) error {
	fileEntries, err := os.ReadDir(parentDir)
	if err != nil {
		return err
	}

	for _, entry := range fileEntries {
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}

		if util.IsDirOrSymLinkToDir(parentDir, fileInfo) {
			err = handleDir(zipWriter, baseDir, parentDir, filters, includeFilters, fileInfo)
		} else {
			err = handleFile(zipWriter, baseDir, parentDir, filters, includeFilters, fileInfo)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func handleFile(
	zipWriter *zip.Writer,
	baseDir,
	parentDir string,
	filters,
	includeFilters []string,
	file fs.FileInfo,
) error {
	fileName := parentDir + file.Name()
	if filterMatched(includeFilters, file.Name()) && filterMatched(filters, file.Name()) {
		logger.PrintIfVerbose("Included: " + fileName)
		dat, err := ioutil.ReadFile(parentDir + file.Name())
		if err != nil {
			if os.IsNotExist(err) {
				logger.PrintfIfVerbose("%s: %s: %v", DanglingSymlinkError, fileName, err)
				return nil
			}
			return err
		}
		f, err := zipWriter.Create(baseDir + file.Name())
		if err != nil {
			return err
		}
		_, err = f.Write(dat)
		if err != nil {
			return err
		}
	} else {
		logger.PrintIfVerbose("Excluded: " + fileName)
	}
	return nil
}

func handleDir(
	zipWriter *zip.Writer,
	baseDir,
	parentDir string,
	filters,
	includeFilters []string,
	file fs.FileInfo,
) error {
	// Check if folder belongs to the disabled exclusions
	if commonParams.DisabledExclusions[file.Name()] {
		logger.PrintIfVerbose("The folder " + file.Name() + " is being included")
		newParent, newBase := GetNewParentAndBase(parentDir, file, baseDir)
		return addDirFilesIgnoreFilter(zipWriter, newBase, newParent)
	}

	isFiltered, err := isDirFiltered(file.Name(), filters)
	if err != nil {
		return err
	}
	if isFiltered {
		logger.PrintIfVerbose("Excluded: " + parentDir + file.Name() + "/")
		return nil
	}
	newParent, newBase := GetNewParentAndBase(parentDir, file, baseDir)
	return addDirFiles(zipWriter, newBase, newParent, filters, includeFilters)
}

func isDirFiltered(filename string, filters []string) (bool, error) {
	for _, filter := range filters {
		if filter[0] == '!' {
			filterStr := strings.TrimSuffix(filepath.ToSlash(filter[1:]), "/")
			match, err := path.Match(filterStr, filename)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
			}
		}
	}

	return false, nil
}

func GetNewParentAndBase(parentDir string, file fs.FileInfo, baseDir string) (newParent, newBase string) {
	logger.PrintIfVerbose("Directory: " + parentDir + file.Name())
	newParent = parentDir + file.Name() + "/"
	newBase = baseDir + file.Name() + "/"
	return newParent, newBase
}

func filterMatched(filters []string, fileName string) bool {
	firstMatch := true
	matched := true
	for _, filter := range filters {
		if filter[0] == '!' {
			// it just needs to match one exclusion to be excluded.
			excluded, _ := path.Match(filter[1:], fileName)
			if excluded {
				return false
			}
		} else {
			// If there are no inclusions everything is considered included
			if firstMatch {
				// Inclusion filter found so reset the match flag and search for a match
				firstMatch = false
				matched = false
			}
			// We can't immediately return as we can still find an exclusion further down the slice
			// So we store the match result and never try again
			if !matched {
				matched, _ = path.Match(filter, fileName)
			}
		}
	}
	return matched
}

func runScaResolver(sourceDir, scaResolver, scaResolverParams, projectName string) error {
	if scaResolver != "" {
		scaFile, err := ioutil.TempFile("", "sca")
		scaResolverResultsFile = scaFile.Name() + ".json"
		if err != nil {
			return err
		}
		args := []string{
			"offline",
			"-s",
			sourceDir,
			"-n",
			projectName,
			"-r",
			scaResolverResultsFile,
		}
		if scaResolverParams != "" {
			parsedscaResolverParams := parseArgs(scaResolverParams)
			args = append(args, parsedscaResolverParams...)
		}
		log.Println(fmt.Sprintf("Using SCA resolver: %s %v", scaResolver, args))
		out, err := exec.Command(scaResolver, args...).Output()
		logger.PrintIfVerbose(string(out))
		if err != nil {
			return errors.Errorf("%s", err)
		}
	}
	return nil
}

func addScaResults(zipWriter *zip.Writer) error {
	logger.PrintIfVerbose("Included SCA Results: " + ".cxsca-results.json")
	dat, err := ioutil.ReadFile(scaResolverResultsFile)
	scaResultsFile := strings.TrimSuffix(scaResolverResultsFile, ".json")
	_ = os.Remove(scaResolverResultsFile)
	if err != nil {
		return err
	}
	removeErr := os.Remove(scaResultsFile)
	if removeErr != nil {
		log.Printf("Failed to remove file %s: %v", scaResultsFile, removeErr)
	} else {
		log.Printf("Successfully removed file %s", scaResultsFile)
	}
	f, err := zipWriter.Create(".cxsca-results.json")
	if err != nil {
		return err
	}
	_, err = f.Write(dat)
	if err != nil {
		return err
	}
	return nil
}

func getUploadURLFromSource(cmd *cobra.Command, uploadsWrapper wrappers.UploadsWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) (
	url, zipFilePath string, err error) {
	var preSignedURL string

	sourceDirFilter, _ := cmd.Flags().GetString(commonParams.SourceDirFilterFlag)
	userIncludeFilter, _ := cmd.Flags().GetString(commonParams.IncludeFilterFlag)
	projectName, _ := cmd.Flags().GetString(commonParams.ProjectName)
	containerScanTriggered := strings.Contains(actualScanTypes, commonParams.ContainersType)
	containerImagesFlag, _ := cmd.Flags().GetString(commonParams.ContainerImagesFlag)
	containerResolveLocally, _ := cmd.Flags().GetBool(commonParams.ContainerResolveLocallyFlag)
	scaResolverPath, _ := cmd.Flags().GetString(commonParams.ScaResolverFlag)

	scaResolverParams, scaResolver := getScaResolverFlags(cmd)
	isSbom, _ := cmd.PersistentFlags().GetBool(commonParams.SbomFlag)
	var directoryPath string
	if isSbom {
		sbomFile, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		isValid, err := isValidJSONOrXML(sbomFile)
		if err != nil {
			return "", "", errors.Wrapf(err, "%s: Input in bad format", failedCreating)
		}
		if !isValid {
			return "", "", errors.Wrapf(err, "%s: Input in bad format", failedCreating)
		}
		zipFilePath, err = util.CompressFile(sbomFile, "sbomFileCompress", directoryCreationPrefix)
		if err != nil {
			return "", "", errors.Wrapf(err, "%s: Input in bad format", failedCreating)
		}
	} else {
		zipFilePath, directoryPath, err = definePathForZipFileOrDirectory(cmd)
	}

	if zipFilePath != "" && scaResolverPath != "" {
		return "", "", errors.New("Scanning Zip files is not supported by ScaResolver.Please use non-zip source")
	}
	if err != nil {
		return "", "", errors.Wrapf(err, "%s: Input in bad format", failedCreating)
	}

	var errorUnzippingFile error
	userProvidedZip := len(zipFilePath) > 0

	unzip := ((len(sourceDirFilter) > 0 || len(userIncludeFilter) > 0) || containerScanTriggered) && userProvidedZip
	if unzip {
		directoryPath, errorUnzippingFile = UnzipFile(zipFilePath)
		if errorUnzippingFile != nil {
			return "", "", errorUnzippingFile
		}
	}

	if directoryPath != "" {
		var dirPathErr error

		// execute scaResolver only in sca type of scans
		if strings.Contains(actualScanTypes, commonParams.ScaType) {
			scaErr := runScaResolver(directoryPath, scaResolver, scaResolverParams, projectName)
			if scaErr != nil {
				if unzip {
					_ = cleanTempUnzipDirectory(directoryPath)
				}
				return "", "", errors.Wrapf(scaErr, "ScaResolver error")
			}
		}

		if containerScanTriggered && containerResolveLocally {
			containerResolverError := runContainerResolver(cmd, directoryPath, containerImagesFlag, containerResolveLocally)
			if containerResolverError != nil {
				if unzip {
					_ = cleanTempUnzipDirectory(directoryPath)
				}
				return "", "", containerResolverError
			}
		}

		if isSingleContainerScanTriggered() && containerResolveLocally {
			logger.PrintIfVerbose("Single container scan triggered: compressing only the container resolution file")
			containerResolutionFilePath := filepath.Join(directoryPath, ".checkmarx", "containers", containerResolutionFileName)
			zipFilePath, dirPathErr = util.CompressFile(containerResolutionFilePath, containerResolutionFileName, directoryCreationPrefix)
		} else if isSingleContainerScanTriggered() && containerImagesFlag != "" {
			logger.PrintIfVerbose("Single container scan with external images: creating minimal zip file")
			// For container scans with external images, we need to create a minimal zip file
			// since the API requires an upload URL even for container-only scans
			zipFilePath, dirPathErr = createMinimalZipFile()
			if unzip {
				dirRemovalErr := cleanTempUnzipDirectory(directoryPath)
				if dirRemovalErr != nil {
					return "", "", dirRemovalErr
				}
			}
		} else {
			if !isSbom {
				zipFilePath, dirPathErr = compressFolder(directoryPath, sourceDirFilter, userIncludeFilter, scaResolver)
			}
		}
		if dirPathErr != nil {
			return "", "", dirPathErr
		}

		if unzip {
			dirRemovalErr := cleanTempUnzipDirectory(directoryPath)
			if dirRemovalErr != nil {
				return "", "", dirRemovalErr
			}
		}
	}

	if zipFilePath != "" && !isSbom {
		return uploadZip(uploadsWrapper, zipFilePath, unzip, userProvidedZip, featureFlagsWrapper)
	} else if zipFilePath != "" && isSbom {
		return uploadZip(uploadsWrapper, zipFilePath, unzip, false, featureFlagsWrapper)
	}
	return preSignedURL, zipFilePath, nil
}

func runContainerResolver(cmd *cobra.Command, directoryPath, containerImageFlag string, containerResolveLocally bool) error {
	debug, _ := cmd.Flags().GetBool(commonParams.DebugFlag)
	var containerImagesList []string

	if containerImageFlag != "" {
		containerImagesList = strings.Split(strings.TrimSpace(containerImageFlag), ",")
		for _, containerImageName := range containerImagesList {
			if containerImagesErr := validateContainerImageFormat(containerImageName); containerImagesErr != nil {
				return containerImagesErr
			}
		}
		logger.PrintIfVerbose(fmt.Sprintf("User input container images identified: %v", strings.Join(containerImagesList, ", ")))
	}
	if containerResolveLocally || len(containerImagesList) > 0 {
		containerResolverErr := containerResolver.Resolve(directoryPath, directoryPath, containerImagesList, debug)
		if containerResolverErr != nil {
			return containerResolverErr
		}
	}
	return nil
}

func uploadZip(uploadsWrapper wrappers.UploadsWrapper, zipFilePath string, unzip, userProvidedZip bool, featureFlagsWrapper wrappers.FeatureFlagsWrapper) (
	url, zipPath string,
	err error,
) {
	var zipFilePathErr error
	// Send a request to uploads service
	var preSignedURL *string
	preSignedURL, zipFilePathErr = uploadsWrapper.UploadFile(zipFilePath, featureFlagsWrapper)
	if zipFilePathErr != nil {
		if unzip || !userProvidedZip {
			return "", zipFilePath, errors.Wrapf(zipFilePathErr, "%s: Failed to upload sources file\n", failedCreating)
		}
		return "", "", errors.Wrapf(zipFilePathErr, "%s: Failed to upload sources file\n", failedCreating)
	}
	if unzip || !userProvidedZip {
		return *preSignedURL, zipFilePath, zipFilePathErr
	}
	return *preSignedURL, "", zipFilePathErr
}

func getScaResolverFlags(cmd *cobra.Command) (scaResolverParams, scaResolver string) {
	scaResolverParams, dirPathErr := cmd.Flags().GetString(commonParams.ScaResolverParamsFlag)
	if dirPathErr != nil {
		scaResolverParams = ""
	}
	scaResolver, dirPathErr = cmd.Flags().GetString(commonParams.ScaResolverFlag)
	if dirPathErr != nil {
		scaResolver = ""
		scaResolverParams = ""
	}
	logger.PrintfIfVerbose("Sca-Resolver params:: %v", scaResolverParams)
	return scaResolverParams, scaResolver
}

func UnzipFile(f string) (string, error) {
	tempDir := filepath.Join(os.TempDir(), "cx-unzipped-temp-dir-") + uuid.New().String() + string(os.PathSeparator)

	err := os.Mkdir(tempDir, directoryPermission)
	if err != nil {
		return "", errors.Errorf("%s %s", errorUnzippingFile, err.Error())
	}

	archive, err := zip.OpenReader(f)
	if err != nil {
		return "", errors.Errorf("%s %s", errorUnzippingFile, err.Error())
	}
	defer func() {
		_ = archive.Close()
	}()
	for _, f := range archive.File {
		filePath := filepath.Join(tempDir, f.Name)
		logger.PrintIfVerbose("unzipping file " + filePath + "...")

		if !strings.HasPrefix(filePath, filepath.Clean(tempDir)+string(os.PathSeparator)) {
			return "", errors.New("invalid file path " + filePath)
		}
		if f.FileInfo().IsDir() {
			logger.PrintIfVerbose("creating directory...")
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return "", errors.Errorf("%s %s", errorUnzippingFile, err.Error())
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return "", errors.Errorf("%s %s", errorUnzippingFile, err.Error())
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", errors.Errorf("%s %s", errorUnzippingFile, err.Error())
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return "", errors.Errorf("%s %s", errorUnzippingFile, err.Error())
		}

		if _, err = io.Copy(dstFile, fileInArchive); err != nil {
			return "", errors.Errorf("%s %s", errorUnzippingFile, err.Error())
		}

		err = dstFile.Close()
		if err != nil {
			return "", errors.Errorf("%s %s", errorUnzippingFile, err.Error())
		}
		err = fileInArchive.Close()
		if err != nil {
			return "", errors.Errorf("%s %s", errorUnzippingFile, err.Error())
		}
	}

	return tempDir, nil
}

func definePathForZipFileOrDirectory(cmd *cobra.Command) (zipFile, sourceDir string, err error) {
	source, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
	sourceTrimmed := strings.TrimSpace(source)

	info, statErr := os.Stat(sourceTrimmed)
	if !os.IsNotExist(statErr) {
		if filepath.Ext(sourceTrimmed) == constants.ZipExtension {
			zipFile = sourceTrimmed
		} else if info != nil && info.IsDir() {
			sourceDir = filepath.ToSlash(sourceTrimmed)
			if !strings.HasSuffix(sourceDir, "/") {
				sourceDir += "/"
			}
		} else {
			msg := fmt.Sprintf("Sources input has bad format: %v", sourceTrimmed)
			err = errors.New(msg)
		}
	} else {
		msg := fmt.Sprintf("Sources input has bad format: %v", sourceTrimmed)
		err = errors.New(msg)
	}

	return zipFile, sourceDir, err
}

func runCreateScanCommand(
	scansWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	jwtWrapper wrappers.JWTWrapper,
	policyWrapper wrappers.PolicyWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := validateScanTypes(cmd, jwtWrapper, featureFlagsWrapper)
		if err != nil {
			return err
		}
		err = validateCreateScanFlags(cmd)
		if err != nil {
			return err
		}
		timeoutMinutes, _ := cmd.Flags().GetInt(commonParams.ScanTimeoutFlag)
		if timeoutMinutes < 0 {
			return errors.Errorf("--%s should be equal or higher than 0", commonParams.ScanTimeoutFlag)
		}
		threshold, _ := cmd.Flags().GetString(commonParams.Threshold)
		thresholdMap := parseThreshold(threshold)
		err = validateThresholds(thresholdMap)
		if err != nil {
			return err
		}
		scanModel, zipFilePath, err := createScanModel(
			cmd,
			uploadsWrapper,
			projectsWrapper,
			groupsWrapper,
			scansWrapper,
			accessManagementWrapper,
			applicationsWrapper,
			featureFlagsWrapper,
			jwtWrapper,
		)
		defer cleanUpTempZip(zipFilePath)
		if err != nil {
			return errors.Errorf("%s", err)
		}
		scanResponseModel, errorModel, err := scansWrapper.Create(scanModel)
		if err != nil {
			return errors.Wrapf(err, "%s", failedCreating)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedCreating, errorModel.Code, errorModel.Message)
		} else if scanResponseModel != nil {
			scanResponseModel = enrichScanResponseModel(cmd, scanResponseModel)
			err = printByScanInfoFormat(cmd, toScanView(scanResponseModel))
			if err != nil {
				return errors.Wrapf(err, "%s\n", failedCreating)
			}
		}
		// Wait until the scan is done: Queued, Running
		AsyncFlag, _ := cmd.Flags().GetBool(commonParams.AsyncFlag)
		policyResponseModel := &wrappers.PolicyResponseModel{}
		if !AsyncFlag {
			waitDelay, _ := cmd.Flags().GetInt(commonParams.WaitDelayFlag)
			err = handleWait(
				cmd,
				scanResponseModel,
				waitDelay,
				timeoutMinutes,
				scansWrapper,
				exportWrapper,
				resultsPdfReportsWrapper,
				resultsJSONReportsWrapper,
				resultsWrapper,
				risksOverviewWrapper,
				scsScanOverviewWrapper,
				featureFlagsWrapper)
			if err != nil {
				return err
			}

			agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)
			policyTimeout, _ := cmd.Flags().GetInt(commonParams.PolicyTimeoutFlag)
			policyResponseModel, err = services.HandlePolicyEvaluation(cmd, policyWrapper, scanResponseModel, agent, waitDelay, policyTimeout)
			if err != nil {
				return err
			}

			results, reportErr := createReportsAfterScan(cmd, scanResponseModel.ID, scansWrapper, exportWrapper, resultsPdfReportsWrapper, resultsJSONReportsWrapper,
				resultsWrapper, risksOverviewWrapper, scsScanOverviewWrapper, policyResponseModel, featureFlagsWrapper)
			if reportErr != nil {
				return reportErr
			}

			err = applyThreshold(cmd, scanResponseModel, thresholdMap, risksOverviewWrapper, results)

			if err != nil {
				return err
			}
		} else {
			_, err = createReportsAfterScan(cmd, scanResponseModel.ID, scansWrapper, exportWrapper, resultsPdfReportsWrapper, resultsJSONReportsWrapper, resultsWrapper,
				risksOverviewWrapper, scsScanOverviewWrapper, nil, featureFlagsWrapper)
			if err != nil {
				return err
			}
		}

		// verify break build from policy
		if policyResponseModel != nil && len(policyResponseModel.Policies) > 0 && policyResponseModel.BreakBuild {
			logger.PrintIfVerbose("Breaking the build due to policy violation")
			return errors.Errorf("Policy Violation - Break Build Enabled.")
		}
		return nil
	}
}

func enrichScanResponseModel(
	cmd *cobra.Command, scanResponseModel *wrappers.ScanResponseModel,
) *wrappers.ScanResponseModel {
	scanResponseModel.ProjectName, _ = cmd.Flags().GetString(commonParams.ProjectName)
	incrementalSast, _ := cmd.Flags().GetBool(commonParams.IncrementalSast)
	scanResponseModel.SastIncremental = strconv.FormatBool(incrementalSast)
	timeoutVal, _ := cmd.Flags().GetInt(commonParams.ScanTimeoutFlag)
	scanResponseModel.Timeout = strconv.Itoa(timeoutVal)
	return scanResponseModel
}

func createScanModel(
	cmd *cobra.Command,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	scansWrapper wrappers.ScansWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	jwtWrapper wrappers.JWTWrapper,
) (*wrappers.Scan, string, error) {
	var input = []byte("{}")

	// Define type, project and config in scan model
	err := setupScanTypeProjectAndConfig(&input, cmd, projectsWrapper, groupsWrapper, scansWrapper, applicationsWrapper, accessManagementWrapper, featureFlagsWrapper, jwtWrapper)
	if err != nil {
		return nil, "", err
	}

	// set tags in scan model
	setupScanTags(&input, cmd)

	scanModel := wrappers.Scan{}
	// Try to parse to a scan model in order to manipulate the request payload
	err = json.Unmarshal(input, &scanModel)
	if err != nil {
		return nil, "", errors.Wrapf(err, "%s: Input in bad format", failedCreating)
	}

	// Set up the scan handler (either git or upload)
	scanHandler, zipFilePath, err := setupScanHandler(cmd, uploadsWrapper, featureFlagsWrapper)
	if err != nil {
		return nil, zipFilePath, err
	}

	scanModel.Handler, _ = json.Marshal(scanHandler)

	uploadType := getUploadType(cmd)

	if uploadType == "git" {
		log.Printf("\n\nScanning branch %s...\n", viper.GetString(commonParams.BranchKey))
	}

	return &scanModel, zipFilePath, nil
}

func getUploadType(cmd *cobra.Command) string {
	source, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
	sourceTrimmed := strings.TrimSpace(source)

	if util.IsGitURL(sourceTrimmed) {
		return git
	}

	return "upload"
}

func setupScanHandler(cmd *cobra.Command, uploadsWrapper wrappers.UploadsWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) (
	wrappers.ScanHandler,
	string,
	error,
) {
	zipFilePath := ""
	scanHandler := wrappers.ScanHandler{}
	scanHandler.Branch = strings.TrimSpace(viper.GetString(commonParams.BranchKey))

	uploadType := getUploadType(cmd)

	if uploadType == git {
		source, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		scanHandler.RepoURL = strings.TrimSpace(source)
	} else {
		var err error
		var uploadURL string
		uploadURL, zipFilePath, err = getUploadURLFromSource(cmd, uploadsWrapper, featureFlagsWrapper)
		if err != nil {
			return scanHandler, zipFilePath, err
		}

		scanHandler.UploadURL = uploadURL
	}
	logger.Print("Temporary zip file path: " + zipFilePath)

	var err error

	// Define SSH credentials if flag --ssh-key is provided
	if cmd.Flags().Changed(commonParams.SSHKeyFlag) {
		sshKeyPath, _ := cmd.Flags().GetString(commonParams.SSHKeyFlag)

		if strings.TrimSpace(sshKeyPath) == "" {
			return scanHandler, "", errors.New("flag needs an argument: --ssh-key")
		}

		source, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		sourceTrimmed := strings.TrimSpace(source)

		if !util.IsSSHURL(sourceTrimmed) {
			return scanHandler, "", errors.New(invalidSSHSource)
		}

		err = defineSSHCredentials(strings.TrimSpace(sshKeyPath), &scanHandler)
	}

	return scanHandler, zipFilePath, err
}

func defineSSHCredentials(sshKeyPath string, handler *wrappers.ScanHandler) error {
	sshKey, err := util.ReadFileAsString(sshKeyPath)
	if err != nil {
		return err
	}
	viper.Set(commonParams.SSHValue, sshKey)

	credentials := wrappers.GitCredentials{}

	credentials.Type = "ssh"
	credentials.Value = sshKey

	handler.Credentials = credentials

	return nil
}

func handleWait(
	cmd *cobra.Command,
	scanResponseModel *wrappers.ScanResponseModel,
	waitDelay,
	timeoutMinutes int,
	scansWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) error {
	err := waitForScanCompletion(
		scanResponseModel,
		waitDelay,
		timeoutMinutes,
		scansWrapper,
		exportWrapper,
		resultsPdfReportsWrapper,
		resultsJSONReportsWrapper,
		resultsWrapper,
		risksOverviewWrapper,
		scsScanOverviewWrapper,
		cmd,
		featureFlagsWrapper)
	if err != nil {
		verboseFlag, _ := cmd.Flags().GetBool(commonParams.DebugFlag)
		if verboseFlag {
			log.Println("Printing workflow logs")
			taskResponseModel, _, _ := scansWrapper.GetWorkflowByID(scanResponseModel.ID)
			_ = printer.Print(cmd.OutOrStdout(), taskResponseModel, printer.FormatList)
		}
		return err
	}
	return nil
}

func createReportsAfterScan(
	cmd *cobra.Command,
	scanID string,
	scansWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	policyResponseModel *wrappers.PolicyResponseModel,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) (*wrappers.ScanResultsCollection, error) {
	// Create the required reports
	targetFile, _ := cmd.Flags().GetString(commonParams.TargetFlag)
	targetPath, _ := cmd.Flags().GetString(commonParams.TargetPathFlag)
	reportFormats, _ := cmd.Flags().GetString(commonParams.TargetFormatFlag)
	formatPdfToEmail, _ := cmd.Flags().GetString(commonParams.ReportFormatPdfToEmailFlag)
	formatPdfOptions, _ := cmd.Flags().GetString(commonParams.ReportFormatPdfOptionsFlag)
	formatSbomOptions, _ := cmd.Flags().GetString(commonParams.ReportSbomFormatFlag)
	agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)
	scaHideDevAndTestDep, _ := cmd.Flags().GetBool(commonParams.ScaHideDevAndTestDepFlag)

	resultsParams, err := getFilters(cmd)
	if err != nil {
		return nil, err
	}

	if scaHideDevAndTestDep {
		resultsParams[ScaExcludeResultTypesParam] = ScaDevAndTestExclusionParam
	}

	if !strings.Contains(reportFormats, printer.FormatSummaryConsole) {
		reportFormats += "," + printer.FormatSummaryConsole
	}
	scan, errorModel, scanErr := scansWrapper.GetByID(scanID)
	if scanErr != nil {
		return nil, errors.Wrapf(scanErr, "%s", failedGetting)
	}
	if errorModel != nil {
		return nil, errors.Errorf("%s: CODE: %d, %s", failedGettingScan, errorModel.Code, errorModel.Message)
	}
	return CreateScanReport(
		resultsWrapper,
		risksOverviewWrapper,
		scsScanOverviewWrapper,
		exportWrapper,
		policyResponseModel,
		resultsPdfReportsWrapper,
		resultsJSONReportsWrapper,
		scan,
		reportFormats,
		formatPdfToEmail,
		formatPdfOptions,
		formatSbomOptions,
		targetFile,
		targetPath,
		agent,
		resultsParams,
		featureFlagsWrapper,
	)
}

func applyThreshold(
	cmd *cobra.Command,
	scanResponseModel *wrappers.ScanResponseModel,
	thresholdMap map[string]int,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	results *wrappers.ScanResultsCollection,
) error {
	if len(thresholdMap) == 0 {
		return nil
	}

	sastRedundancy, _ := cmd.Flags().GetBool(commonParams.SastRedundancyFlag)
	params := make(map[string]string)
	if sastRedundancy {
		params[commonParams.SastRedundancyFlag] = ""
	}

	summaryMap, err := getSummaryThresholdMap(scanResponseModel, risksOverviewWrapper, results)

	if err != nil {
		return err
	}

	var errorBuilder strings.Builder
	var messageBuilder strings.Builder
	for key, thresholdLimit := range thresholdMap {
		currentValue := summaryMap[key]
		failed := currentValue >= thresholdLimit
		logMessage := fmt.Sprintf(thresholdLog, key, thresholdLimit, currentValue)
		logger.PrintIfVerbose(logMessage)

		if failed {
			errorBuilder.WriteString(fmt.Sprintf("%s | ", logMessage))
		} else {
			messageBuilder.WriteString(fmt.Sprintf("%s | ", logMessage))
		}
	}

	errorMessage := errorBuilder.String()
	if errorMessage != "" {
		return errors.Errorf(thresholdMsgLog, "Failed", errorMessage)
	}

	successMessage := messageBuilder.String()
	if successMessage != "" {
		log.Printf(thresholdMsgLog, "Success", successMessage)
	}

	return nil
}

func parseThreshold(threshold string) map[string]int {
	if strings.TrimSpace(threshold) == "" {
		return nil
	}
	thresholdMap := make(map[string]int)
	if threshold != "" {
		threshold = strings.ReplaceAll(strings.ReplaceAll(threshold, " ", ""), ",", ";")
		thresholdLimits := strings.Split(strings.ToLower(threshold), ";")
		for _, limits := range thresholdLimits {
			engineName, intLimit, err := parseThresholdLimit(limits)
			if err != nil {
				log.Printf("%s", err)
			} else {
				thresholdMap[engineName] = intLimit
			}
		}
	}

	return thresholdMap
}

func validateThresholds(thresholdMap map[string]int) error {
	var errMsgBuilder strings.Builder

	for engineName, limit := range thresholdMap {
		if limit < 1 {
			errMsgBuilder.WriteString(errors.Errorf("Invalid value for threshold limit %s. Threshold should be greater or equal to 1.\n", engineName).Error())
		}
	}

	errMsg := errMsgBuilder.String()
	if errMsg != "" {
		return errors.New(errMsg)
	}
	return nil
}

func parseThresholdLimit(limit string) (engineName string, intLimit int, err error) {
	parts := strings.Split(limit, "=")
	engineName = strings.Replace(parts[0], commonParams.KicsType, commonParams.IacType, 1)
	if len(parts) <= 1 {
		return engineName, 0, errors.New("Error parsing threshold limit: missing values\n")
	}
	intLimit, err = strconv.Atoi(parts[1])
	if err != nil {
		err = errors.Errorf("%s: Error parsing threshold limit: %v\n", engineName, err)
	}
	return engineName, intLimit, err
}

func getSummaryThresholdMap(
	scan *wrappers.ScanResponseModel,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	results *wrappers.ScanResultsCollection,
) (map[string]int, error) {
	summaryMap := make(map[string]int)

	for _, result := range results.Results {
		if isExploitable(result.State) {
			key := strings.ToLower(fmt.Sprintf("%s-%s", strings.Replace(result.Type, commonParams.KicsType, commonParams.IacType, 1), result.Severity))
			summaryMap[key]++
		}
	}

	if slices.Contains(scan.Engines, commonParams.APISecType) {
		apiSecRisks, err := getResultsForAPISecScanner(risksOverviewWrapper, scan.ID)
		if err != nil {
			return nil, err
		}
		summaryMap["api-security-high"] = apiSecRisks.Risks[1]
		summaryMap["api-security-medium"] = apiSecRisks.Risks[2]
		summaryMap["api-security-low"] = apiSecRisks.Risks[3]
	}
	return summaryMap, nil
}

func isExploitable(state string) bool {
	return !strings.EqualFold(state, notExploitable) && !strings.EqualFold(state, ignored)
}

func waitForScanCompletion(
	scanResponseModel *wrappers.ScanResponseModel,
	waitDelay,
	timeoutMinutes int,
	scansWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	cmd *cobra.Command,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) error {
	log.Println("Wait for scan to complete", scanResponseModel.ID, scanResponseModel.Status)
	timeout := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)
	fixedWait := time.Duration(waitDelay) * time.Second
	i := uint64(0)
	if !cmd.Flags().Changed(commonParams.RetryDelayFlag) {
		viper.Set(commonParams.RetryDelayFlag, commonParams.RetryDelayPollingDefault)
	}
	for {
		variableWait := time.Duration(math.Min(float64(i/uint64(waitDelay)), maxPollingWaitTime)) * time.Second
		waitDuration := fixedWait + variableWait
		logger.PrintfIfVerbose("Sleeping %v before polling", waitDuration)
		time.Sleep(waitDuration)
		running, err := isScanRunning(scansWrapper, exportWrapper, resultsPdfReportsWrapper, resultsJSONReportsWrapper, resultsWrapper,
			risksOverviewWrapper, scsScanOverviewWrapper, scanResponseModel.ID, cmd, featureFlagsWrapper)
		if err != nil {
			return err
		}
		if !running {
			break
		}
		if timeoutMinutes > 0 && time.Now().After(timeout) {
			log.Println("Canceling scan", scanResponseModel.ID)
			errorModel, err := scansWrapper.Cancel(scanResponseModel.ID)
			if err != nil {
				return errors.Wrapf(err, "%s\n", failedCanceling)
			}
			if errorModel != nil {
				return errors.Errorf(services.ErrorCodeFormat, failedCanceling, errorModel.Code, errorModel.Message)
			}

			return wrappers.NewAstError(exitCodes.MultipleEnginesFailedExitCode, errors.Errorf("Timeout of %d minute(s) for scan reached", timeoutMinutes))
		}
		i++
	}
	return nil
}

func isScanRunning(
	scansWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverViewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	scanID string,
	cmd *cobra.Command,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) (bool, error) {
	var scanResponseModel *wrappers.ScanResponseModel
	var errorModel *wrappers.ErrorModel
	var err error
	scanResponseModel, errorModel, err = scansWrapper.GetByID(scanID)
	if err != nil {
		log.Fatal("Cannot source code temp file.", err)
	}
	if errorModel != nil {
		log.Fatalf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message)
	} else if scanResponseModel != nil {
		if scanResponseModel.Status == wrappers.ScanRunning || scanResponseModel.Status == wrappers.ScanQueued {
			log.Println("Scan status: ", scanResponseModel.Status)
			return true, nil
		}
	}
	log.Println("Scan Finished with status: ", scanResponseModel.Status)
	if scanResponseModel.Status == wrappers.ScanPartial {
		_ = printer.Print(cmd.OutOrStdout(), scanResponseModel.StatusDetails, printer.FormatList)
		_, reportErr := createReportsAfterScan(
			cmd,
			scanResponseModel.ID,
			scansWrapper,
			exportWrapper,
			resultsPdfReportsWrapper,
			resultsJSONReportsWrapper,
			resultsWrapper,
			risksOverViewWrapper,
			scsScanOverviewWrapper,
			nil, featureFlagsWrapper) // check this partial case, how to handle it
		if reportErr != nil {
			return false, errors.New("unable to create report for partial scan")
		}
		exitCode := getExitCode(scanResponseModel)
		return false, wrappers.NewAstError(exitCode, errors.New("scan completed partially"))
	} else if scanResponseModel.Status != wrappers.ScanCompleted {
		exitCode := getExitCode(scanResponseModel)
		return false, wrappers.NewAstError(exitCode, errors.New("scan did not complete successfully"))
	}
	return false, nil
}

func getExitCode(scanResponseModel *wrappers.ScanResponseModel) int {
	failedStatuses := make([]int, 0)
	for _, scanner := range scanResponseModel.StatusDetails {
		scannerNameLowerCase := strings.ToLower(scanner.Name)
		scannerErrorExitCode, errorCodeByScannerExists := errorCodesByScanner[scannerNameLowerCase]
		if scanner.Status == wrappers.ScanFailed && scanner.Name != General && errorCodeByScannerExists {
			failedStatuses = append(failedStatuses, scannerErrorExitCode)
		}
	}
	if len(failedStatuses) == 1 {
		return failedStatuses[0]
	}

	return exitCodes.MultipleEnginesFailedExitCode
}

const (
	General     = "general"
	Sast        = "sast"
	Sca         = "sca"
	IacSecurity = "iac-security" // We get 'kics' from AST. Added for forward compatibility
	Kics        = "kics"
	APISec      = "apisec"
)

var errorCodesByScanner = map[string]int{
	General:     exitCodes.MultipleEnginesFailedExitCode,
	Sast:        exitCodes.SastEngineFailedExitCode,
	Sca:         exitCodes.ScaEngineFailedExitCode,
	IacSecurity: exitCodes.IacSecurityEngineFailedExitCode,
	Kics:        exitCodes.KicsEngineFailedExitCode,
	APISec:      exitCodes.ApisecEngineFailedExitCode,
}

func runListScansCommand(scansWrapper wrappers.ScansWrapper, sastMetadataWrapper wrappers.SastMetadataWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allScansModel *wrappers.ScansCollectionResponseModel
		var errorModel *wrappers.ErrorModel
		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingAll)
		}
		allScansModel, errorModel, err = scansWrapper.Get(params)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingAll)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedGettingAll, errorModel.Code, errorModel.Message)
		} else if allScansModel != nil && allScansModel.Scans != nil {
			views, err := toScanViews(allScansModel.Scans, sastMetadataWrapper)
			if err != nil {
				return err
			}
			err = printByFormat(cmd, views)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func runGetScanByIDCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var scanResponseModel *wrappers.ScanResponseModel
		var errorModel *wrappers.ErrorModel
		var err error
		scanID, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)
		if scanID == "" {
			return errors.Errorf("%s: Please provide a scan ID", failedGetting)
		}
		scanResponseModel, errorModel, err = scansWrapper.GetByID(scanID)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGetting)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message)
		} else if scanResponseModel != nil {
			err = printByFormat(cmd, toScanView(scanResponseModel))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func runScanWorkflowByIDCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var taskResponseModel []*wrappers.ScanTaskResponseModel
		var errorModel *wrappers.ErrorModel
		var err error
		scanID, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)
		if scanID == "" {
			return errors.Errorf("Please provide a scan ID")
		}
		taskResponseModel, errorModel, err = scansWrapper.GetWorkflowByID(scanID)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGetting)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message)
		} else if taskResponseModel != nil {
			err = printByFormat(cmd, taskResponseModel)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func runDeleteScanCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanIDs, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)
		if scanIDs == "" {
			return errors.Errorf("%s: Please provide at least one scan ID", failedDeleting)
		}
		for _, scanID := range strings.Split(scanIDs, ",") {
			errorModel, err := scansWrapper.Delete(scanID)
			if err != nil {
				return errors.Wrapf(err, "%s\n", failedDeleting)
			}

			// Checking the response
			if errorModel != nil {
				return errors.Errorf(services.ErrorCodeFormat, failedDeleting, errorModel.Code, errorModel.Message)
			}
		}

		return nil
	}
}

func runCancelScanCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanIDs, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)
		if scanIDs == "" {
			return errors.Errorf("%s: Please provide at least one scan ID", failedCanceling)
		}
		for _, scanID := range strings.Split(scanIDs, ",") {
			errorModel, err := scansWrapper.Cancel(scanID)
			if err != nil {
				return errors.Wrapf(err, "%s\n", failedCanceling)
			}
			// Checking the response
			if errorModel != nil {
				return errors.Errorf(services.ErrorCodeFormat, failedCanceling, errorModel.Code, errorModel.Message)
			}
		}

		return nil
	}
}

func runGetTagsCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var tags map[string][]string
		var errorModel *wrappers.ErrorModel
		var err error
		tags, errorModel, err = scansWrapper.Tags()
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingTags)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingTags, errorModel.Code, errorModel.Message)
		} else if tags != nil {
			var tagsJSON []byte
			tagsJSON, err = json.Marshal(tags)
			if err != nil {
				return errors.Wrapf(err, "%s: failed to serialize scan tags response ", failedGettingTags)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(tagsJSON))
		}
		return nil
	}
}

func runDownloadLogs(logsWrapper wrappers.LogsWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		scanID, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)
		scanType, _ := cmd.Flags().GetString(commonParams.ScanTypeFlag)
		scanType = strings.Replace(scanType, commonParams.IacType, commonParams.KicsType, 1)
		logText, err := logsWrapper.GetLog(scanID, scanType)
		if err != nil {
			return err
		}
		fmt.Print(logText)
		return nil
	}
}

func runKicksRealtime() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		// Create temp location and add it to container volumes
		volumeMap, tempDir, err := createKicsScanEnv(cmd)
		if err != nil {
			return errors.Errorf("%s", err)
		}

		// Run kics container
		err = runKicsScan(cmd, volumeMap, tempDir, aditionalParameters)
		// Removing temporary dir
		logger.PrintIfVerbose(containerFolderRemoving)
		removeErr := os.RemoveAll(tempDir)
		if removeErr != nil {
			logger.PrintIfVerbose(removeErr.Error())
		}
		return err
	}
}

type scanView struct {
	ID              string `format:"name:Scan ID"`
	ProjectID       string `format:"name:Project ID"`
	ProjectName     string `format:"name:Project Name"`
	Status          string
	CreatedAt       time.Time `format:"name:Created at;time:01-02-06 15:04:05"`
	UpdatedAt       time.Time `format:"name:Updated at;time:01-02-06 15:04:05"`
	Branch          string
	Tags            map[string]string
	SastIncremental string `format:"name:Type"`
	Timeout         string
	Initiator       string
	Origin          string
	Engines         []string
}

func toScanViews(scans []wrappers.ScanResponseModel, sastMetadataWrapper wrappers.SastMetadataWrapper) ([]*scanView, error) {
	scanIDs := make([]string, len(scans))
	for i := range scans {
		scanIDs[i] = scans[i].ID
	}

	sastMetadata, err := services.GetSastMetadataByIDs(sastMetadataWrapper, scanIDs)
	if err != nil {
		logger.Printf("error getting sast metadata: %v", err)
		return nil, err
	}

	metadataMap := make(map[string]bool)
	if sastMetadata != nil {
		for i := range sastMetadata.Scans {
			metadataMap[sastMetadata.Scans[i].ScanID] = sastMetadata.Scans[i].IsIncremental
		}
	}
	views := make([]*scanView, len(scans))
	for i := 0; i < len(scans); i++ {
		scans[i].SastIncremental = strconv.FormatBool(metadataMap[scans[i].ID])
		views[i] = toScanView(&scans[i])
	}
	return views, nil
}

func toScanView(scan *wrappers.ScanResponseModel) *scanView {
	var origin string
	var scanType string
	var scanTimeOut string
	if scan.UserAgent != "" {
		ua := user_agent.New(scan.UserAgent)
		name, version := ua.Browser()
		origin = name + " " + version
	}

	if strings.EqualFold(trueString, scan.SastIncremental) {
		scanType = "Incremental"
	} else {
		scanType = "Full"
	}

	intValForTimeout, err := strconv.Atoi(scan.Timeout)

	if err == nil && intValForTimeout > 0 {
		scanTimeOut = fmt.Sprintf("%s %s", scan.Timeout, "mins")
	} else {
		scanTimeOut = "NONE"
	}
	scan.ReplaceMicroEnginesWithSCS()

	return &scanView{
		ID:              scan.ID,
		Status:          string(scan.Status),
		CreatedAt:       scan.CreatedAt,
		UpdatedAt:       scan.UpdatedAt,
		ProjectName:     scan.ProjectName,
		ProjectID:       scan.ProjectID,
		Branch:          scan.Branch,
		Tags:            scan.Tags,
		SastIncremental: scanType,
		Timeout:         scanTimeOut,
		Initiator:       scan.Initiator,
		Origin:          origin,
		Engines:         scan.Engines,
	}
}

func createKicsScanEnv(cmd *cobra.Command) (volumeMap, kicsDir string, err error) {
	kicsDir, err = ioutil.TempDir("", containerTempDirPattern)
	if err != nil {
		return "", "", errors.New(containerCreateFolderError)
	}
	kicsFilePath, _ := cmd.Flags().GetString(commonParams.KicsRealtimeFile)
	if len(kicsFilePath) < 1 {
		return "", "", errors.New(containerFileSourceMissing)
	}
	if !contains(commonParams.KicsBaseFilters, kicsFilePath) {
		return "", "", errors.New(kicsFilePath + containerFileSourceIncompatible)
	}
	kicsFile, err := ioutil.ReadFile(kicsFilePath)
	if err != nil {
		return "", "", errors.New(containerFileSourceError)
	}
	_, file := filepath.Split(kicsFilePath)
	destinationFile := fmt.Sprintf("%s/%s", kicsDir, file)
	err = ioutil.WriteFile(destinationFile, kicsFile, 0666)
	if err != nil {
		return "", "", errors.New(containerWriteFolderError)
	}
	volumeMap = fmt.Sprintf(containerVolumeFormat, kicsDir)
	return volumeMap, kicsDir, nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v != "" && strings.Contains(str, v) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s []string, str string) bool {
	for _, v := range s {
		if strings.EqualFold(str, v) {
			return true
		}
	}
	return false
}

func readKicsResultsFile(tempDir string) (wrappers.KicsResultsCollection, error) {
	// Open and read the file from the temp folder
	var resultsModel wrappers.KicsResultsCollection
	resultsFile := fmt.Sprintf(containerResultsFileFormat, tempDir)
	jsonFile, err := os.Open(resultsFile)
	if err != nil {
		return resultsModel, err
	}
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return resultsModel, err
	}
	// Unmarshal into the object KicsResultsCollection
	err = json.Unmarshal(byteValue, &resultsModel)
	if err != nil {
		return wrappers.KicsResultsCollection{}, err
	}
	return resultsModel, nil
}

func runKicsScan(cmd *cobra.Command, volumeMap, tempDir string, additionalParameters []string) error {
	var errs error
	kicsRunArgs := []string{
		containerRun,
		containerRemove,
		containerVolumeFlag,
		volumeMap,
		containerNameFlag,
		viper.GetString(commonParams.KicsContainerNameKey),
		containerImage,
		containerScan,
		containerScanPathFlag,
		containerScanPath,
		containerScanOutputFlag,
		containerScanOutput,
		containerScanFormatFlag,
		containerScanFormatOutput,
	}
	// join the additional parameters
	if len(additionalParameters) > 0 {
		kicsRunArgs = append(kicsRunArgs, additionalParameters...)
	}
	logger.PrintIfVerbose(containerStarting)
	logger.PrintIfVerbose(containerFormatInfo)
	kicsCmd, _ := cmd.Flags().GetString(commonParams.KicsRealtimeEngine)
	out, err := exec.Command(kicsCmd, kicsRunArgs...).CombinedOutput()
	logger.PrintIfVerbose(string(out))
	var resultsModel wrappers.KicsResultsCollection
	/* 	NOTE: the kics container returns 40 instead of 0 when successful!! This
	definitely an incorrect behavior but the following check gets past it.
	*/
	if err == nil {
		// no results
		if resultsModel.Results == nil {
			resultsModel.Results = []wrappers.KicsQueries{}
		}
		errs = printKicsResults(&resultsModel)
		if errs != nil {
			return errors.Errorf("%s", errs)
		}
		return nil
	}

	if err != nil {
		errorMessage := err.Error()
		extractedErrorCode := errorMessage[strings.LastIndex(errorMessage, " ")+1:]

		if contains(kicsErrorCodes, extractedErrorCode) {
			resultsModel, errs = readKicsResultsFile(tempDir)
			if errs != nil {
				return errors.Errorf("%s", errs)
			}
			errs = printKicsResults(&resultsModel)
			if errs != nil {
				return errors.Errorf("%s", errs)
			}
			return nil
		}
		exitError, hasExistError := err.(*exec.ExitError)
		if hasExistError {
			if exitError.ExitCode() == util.EngineNoRunningCode {
				logger.PrintIfVerbose(errorMessage)
				return errors.Errorf(util.NotRunningEngineMessage)
			}
		} else {
			if strings.Contains(errorMessage, util.InvalidEngineError) || strings.Contains(
				errorMessage,
				util.InvalidEngineErrorWindows,
			) {
				logger.PrintIfVerbose(errorMessage)
				return errors.Errorf(util.InvalidEngineMessage)
			}
		}
		return errors.Errorf("Check container engine state. Failed: %s", errorMessage)
	}

	return nil
}

func printKicsResults(resultsModel *wrappers.KicsResultsCollection) error {
	var resultsJSON []byte
	resultsJSON, errs := json.Marshal(resultsModel)
	if errs != nil {
		return errors.Errorf("%s", errs)
	}
	fmt.Println(string(resultsJSON))
	return nil
}

func cleanTempUnzipDirectory(directoryPath string) error {
	logger.PrintIfVerbose("Cleaning up temporary directory: " + directoryPath)
	return os.RemoveAll(directoryPath)
}

func cleanUpTempZip(zipFilePath string) {
	if zipFilePath != "" {
		logger.PrintIfVerbose("Cleaning up temporary zip: " + zipFilePath)
		tries := cleanupMaxRetries
		for attempt := 1; tries > 0; attempt++ {
			zipRemoveErr := os.Remove(zipFilePath)
			if zipRemoveErr != nil {
				logger.PrintIfVerbose(
					fmt.Sprintf(
						"Failed to remove temporary zip: Attempt %d/%d: %v",
						attempt,
						cleanupMaxRetries,
						zipRemoveErr,
					),
				)
				tries--
				Wait(attempt)
			} else {
				logger.PrintIfVerbose("Removed temporary zip")
				break
			}
		}
		if tries == 0 {
			logger.PrintIfVerbose("Failed to remove temporary zip " + zipFilePath)
		}
	} else {
		logger.PrintIfVerbose("No temporary zip to clean")
	}
}

func Wait(attempt int) {
	// Calculate exponential backoff delay
	waitDuration := time.Duration(cleanupRetryWaitSeconds * (1 << (attempt - 1))) // 2^(attempt-1)
	logger.PrintIfVerbose(fmt.Sprintf("Waiting %d seconds before retrying...", waitDuration))
	time.Sleep(waitDuration * time.Second)
}

func deprecatedFlagValue(cmd *cobra.Command, deprecatedFlagKey, inUseFlagKey string) string {
	flagValue, _ := cmd.Flags().GetString(inUseFlagKey)
	if flagValue == "" {
		flagValue, _ = cmd.Flags().GetString(deprecatedFlagKey)
	}
	return flagValue
}

func validateCreateScanFlags(cmd *cobra.Command) error {
	isSbomScan, _ := cmd.PersistentFlags().GetBool(commonParams.SbomFlag)
	branch := strings.TrimSpace(viper.GetString(commonParams.BranchKey))
	if branch == "" && !isSbomScan {
		return errors.Errorf("%s: Please provide a branch", failedCreating)
	}
	exploitablePath, _ := cmd.Flags().GetString(commonParams.ExploitablePathFlag)
	lastSastScanTime, _ := cmd.Flags().GetString(commonParams.LastSastScanTime)
	exploitablePath = strings.ToLower(exploitablePath)
	if !strings.Contains(strings.ToLower(actualScanTypes), commonParams.SastType) && strings.EqualFold(exploitablePath, trueString) {
		return errors.Errorf("Please to use --sca-exploitable-path flag in SCA, " +
			"you must enable SAST scan type.")
	}
	err := validateBooleanString(exploitablePath)
	if err != nil {
		return errors.Errorf("Invalid value for --sca-exploitable-path flag. The value must be true or false.")
	}

	if lastSastScanTime != "" {
		lsst, sastErr := strconv.Atoi(lastSastScanTime)
		if sastErr != nil || lsst <= 0 {
			return errors.Errorf("Invalid value for --sca-last-sast-scan-time flag. The value must be a positive integer.")
		}
	}
	projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)
	err = validateBooleanString(projectPrivatePackage)
	if err != nil {
		return errors.Errorf("Invalid value for --project-private-package flag. The value must be true or false.")
	}

	if kicsPresetID, _ := cmd.Flags().GetString(commonParams.IacsPresetIDFlag); kicsPresetID != "" {
		if _, err := uuid.Parse(kicsPresetID); err != nil {
			return fmt.Errorf("Invalid value for --%s flag. Must be a valid UUID.", commonParams.IacsPresetIDFlag)
		}
	}
	// check if flag was passed as arg
	isBranchChanged := cmd.Flags().Changed(commonParams.BranchPrimaryFlag)
	if isBranchChanged {
		for _, a := range os.Args[1:] {
			if strings.HasPrefix(a, BranchPrimaryPrefix) {
				return fmt.Errorf("invalid value for --branch-primary flag. This flag is sent without any values")
			}
		}
	}

	return nil
}

func validateContainerImageFormat(containerImage string) error {
	if strings.HasSuffix(containerImage, ".tar") {
		_, err := osinstaller.FileExists(containerImage)
		if err != nil {
			return errors.Errorf("--container-images flag error: %v", err)
		}

		return nil
	}

	imageParts := strings.Split(containerImage, ":")
	if len(imageParts) != 2 || imageParts[0] == "" || imageParts[1] == "" {
		return errors.Errorf("Invalid value for --container-images flag. The value must be in the format <image-name>:<image-tag> or <image-name>.tar")
	}
	return nil
}

func validateBooleanString(value string) error {
	if value == "" {
		return nil
	}
	lowedValue := strings.ToLower(value)
	if lowedValue != trueString && lowedValue != falseString {
		return errors.Errorf("Invalid value. The value must be true or false.")
	}
	return nil
}

func parseArgs(input string) []string {
	var args []string
	var current strings.Builder
	var quote rune
	inQuotes := false

	for i, r := range input {
		switch {
		case (r == '\'' || r == '"') && !inQuotes:
			inQuotes = true
			quote = r
		case r == quote && inQuotes:
			inQuotes = false
		case unicode.IsSpace(r) && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}

		// Append last token if input ends
		if i == len(input)-1 && current.Len() > 0 {
			args = append(args, current.String())
		}
	}

	return args
}

// createMinimalZipFile creates a minimal zip file for container scans with external images
// The API requires an upload URL even for container-only scans, so we create a minimal placeholder
func createMinimalZipFile() (string, error) {
	outputFile, err := os.CreateTemp(os.TempDir(), "cx-container-*.zip")
	if err != nil {
		return "", errors.Wrapf(err, "Cannot create temp file for container-only scan")
	}
	defer outputFile.Close()

	zipWriter := zip.NewWriter(outputFile)
	defer zipWriter.Close()

	// Create a minimal placeholder file
	f, err := zipWriter.Create(".container-scan")
	if err != nil {
		return "", errors.Wrapf(err, "Cannot create placeholder file in zip")
	}

	// Write minimal content (just a single byte)
	_, err = f.Write([]byte("1"))
	if err != nil {
		return "", errors.Wrapf(err, "Cannot write to placeholder file")
	}

	return outputFile.Name(), nil
}

func isValidJSONOrXML(path string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != jsonExt && ext != xmlExt {
		return false, fmt.Errorf("not a JSON/XML file, provide valid JSON/XMl file")
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	switch ext {
	case jsonExt:
		var js interface{}
		if err := json.Unmarshal(data, &js); err != nil {
			return false, fmt.Errorf("invalid JSON format. %w", err) // Invalid JSON
		}
	case xmlExt:
		var x interface{}
		if err := xml.Unmarshal(data, &x); err != nil {
			return false, fmt.Errorf("invalid XML format.%w", err) // Invalid XML
		}
	default:
		return false, nil
	}

	return true, nil
}
