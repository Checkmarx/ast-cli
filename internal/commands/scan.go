package commands

import (
	"archive/zip"
	"encoding/json"
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
	"strconv"
	"strings"
	"time"

	errorconsts "github.com/checkmarx/ast-cli/internal/errors"

	"github.com/checkmarx/ast-cli/internal/commands/scarealtime"
	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/google/shlex"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/policymanagement"
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

	git                             = "git"
	invalidSSHSource                = "provided source does not need a key. Make sure you are defining the right source or remove the flag --ssh-key"
	errorUnzippingFile              = "an error occurred while unzipping file. Reason: "
	containerRun                    = "run"
	containerVolumeFlag             = "-v"
	containerNameFlag               = "--name"
	containerRemove                 = "--rm"
	containerImage                  = "checkmarx/kics:latest"
	containerScan                   = "scan"
	containerScanPathFlag           = "-p"
	containerScanPath               = "/path"
	containerScanOutputFlag         = "-o"
	containerScanOutput             = "/path"
	containerScanFormatFlag         = "--report-formats"
	containerScanFormatOutput       = "json"
	containerStarting               = "Starting kics container"
	containerFormatInfo             = "The report format and output path cannot be overridden."
	containerFolderRemoving         = "Removing folder in temp"
	containerCreateFolderError      = "Error creating temporary directory"
	containerWriteFolderError       = " Error writing file to temporary directory"
	containerFileSourceMissing      = "--file is required for kics-realtime command"
	containerFileSourceIncompatible = ". Provided file is not supported by kics"
	containerFileSourceError        = " Error reading file"
	containerResultsFileFormat      = "%s/results.json"
	containerVolumeFormat           = "%s:/path"
	containerTempDirPattern         = "kics"
	kicsContainerPrefixName         = "cli-kics-realtime-"
	cleanupMaxRetries               = 3
	cleanupRetryWaitSeconds         = 15
	DanglingSymlinkError            = "Skipping dangling symbolic link"
	configFilterKey                 = "filter"
	configFilterPlatforms           = "platforms"
	configIncremental               = "incremental"
	configPresetName                = "presetName"
	configEngineVerbose             = "engineVerbose"
	configLanguageMode              = "languageMode"
	resultsMapValue                 = "value"
	resultsMapType                  = "type"
	trueString                      = "true"
	falseString                     = "false"
	maxPollingWaitTime              = 60
	engineNotAllowed                = "It looks like the \"%s\" scan type does not exist or you are trying to run a scan without the \"%s\" package license." +
		"\nTo use this feature, you would need to purchase a license." +
		"\nPlease contact our support team for assistance if you believe you have already purchased a license." +
		"\nLicensed packages: %s"
	completedPolicy  = "COMPLETED"
	nonePolicy       = "NONE"
	evaluatingPolicy = "EVALUATING"
)

var (
	scaResolverResultsFile  = ""
	actualScanTypes         = "sast,kics,sca"
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
	kicsErrorCodes      = []string{"50", "40", "30", "20"}
)

func NewScanCommand(
	applicationsWrapper wrappers.ApplicationsWrapper,
	scansWrapper wrappers.ScansWrapper,
	resultsSbomWrapper wrappers.ResultsSbomWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	logsWrapper wrappers.LogsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	riskOverviewWrapper wrappers.RisksOverviewWrapper,
	jwtWrapper wrappers.JWTWrapper,
	scaRealTimeWrapper wrappers.ScaRealTimeWrapper,
	policyWrapper wrappers.PolicyWrapper,
	sastMetadataWrapper wrappers.SastMetadataWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage scans",
		Long:  "The scan command enables the ability to manage scans in Checkmarx One.",
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
		resultsSbomWrapper,
		resultsPdfReportsWrapper,
		uploadsWrapper,
		resultsWrapper,
		projectsWrapper,
		groupsWrapper,
		riskOverviewWrapper,
		jwtWrapper,
		policyWrapper,
		accessManagementWrapper,
		applicationsWrapper,
	)

	listScansCmd := scanListSubCommand(scansWrapper, sastMetadataWrapper)

	showScanCmd := scanShowSubCommand(scansWrapper)

	workflowScanCmd := scanWorkflowSubCommand(scansWrapper)

	deleteScanCmd := scanDeleteSubCommand(scansWrapper)

	cancelScanCmd := scanCancelSubCommand(scansWrapper)

	tagsCmd := scanTagsSubCommand(scansWrapper)

	logsCmd := scanLogsSubCommand(logsWrapper)

	kicsRealtimeCmd := scanRealtimeSubCommand()

	scaRealtimeCmd := scarealtime.NewScaRealtimeCommand(scaRealTimeWrapper)

	addFormatFlagToMultipleCommands(
		[]*cobra.Command{listScansCmd, showScanCmd, workflowScanCmd},
		printer.FormatTable, printer.FormatList, printer.FormatJSON,
	)
	addScanInfoFormatFlag(
		createScanCmd, printer.FormatList, printer.FormatTable, printer.FormatJSON,
	)
	scanCmd.AddCommand(
		createScanCmd,
		showScanCmd,
		workflowScanCmd,
		listScansCmd,
		deleteScanCmd,
		cancelScanCmd,
		tagsCmd,
		logsCmd,
		kicsRealtimeCmd,
		scaRealtimeCmd,
	)
	return scanCmd
}

func scanRealtimeSubCommand() *cobra.Command {
	kicsContainerID := uuid.New()
	viper.Set(commonParams.KicsContainerNameKey, kicsContainerPrefixName+kicsContainerID.String())
	realtimeScanCmd := &cobra.Command{
		Use:   "kics-realtime",
		Short: "Create and run kics scan",
		Long:  "The kics-realtime command enables the ability to create, run and retrieve results from a kics scan using a docker image.",
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
	logsCmd.PersistentFlags().String(commonParams.ScanTypeFlag, "", "Scan type to pull log for, ex: sast, iac-security.")
	markFlagAsRequired(logsCmd, commonParams.ScanIDFlag)
	markFlagAsRequired(logsCmd, commonParams.ScanTypeFlag)

	return logsCmd
}

func scanTagsSubCommand(scansWrapper wrappers.ScansWrapper) *cobra.Command {
	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "Get a list of all available tags to filter by",
		Long:  "The tags command enables the ability to provide a list of all the available tags in Checkmarx One.",
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
		Long:  "The cancel command enables the ability to cancel one or more running scans in Checkmarx One.",
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
		Long:  "The workflow command enables the ability to provide information about a requested scan workflow in Checkmarx One.",
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
	addScanIDFlag(workflowScanCmd, "Scan ID to workflow.")
	return workflowScanCmd
}

func scanShowSubCommand(scansWrapper wrappers.ScansWrapper) *cobra.Command {
	showScanCmd := &cobra.Command{
		Use:   "show",
		Short: "Show information about a scan",
		Long:  "The show command enables the ability to show information about a requested scan in Checkmarx One.",
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
	addScanIDFlag(showScanCmd, "Scan ID to show.")
	return showScanCmd
}

func scanListSubCommand(scansWrapper wrappers.ScansWrapper, sastMetadataWrapper wrappers.SastMetadataWrapper) *cobra.Command {
	listScansCmd := &cobra.Command{
		Use:   "list",
		Short: "List all scans in Checkmarx One",
		Long:  "The list command provides a list of all the scans in Checkmarx One.",
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
	resultsSbomWrapper wrappers.ResultsSbomWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	jwtWrapper wrappers.JWTWrapper,
	policyWrapper wrappers.PolicyWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
) *cobra.Command {
	createScanCmd := &cobra.Command{
		Use:   "create",
		Short: "Create and run a new scan",
		Long:  "The create command enables the ability to create and run a new scan in Checkmarx One.",
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
			resultsSbomWrapper,
			resultsPdfReportsWrapper,
			uploadsWrapper,
			resultsWrapper,
			projectsWrapper,
			groupsWrapper,
			risksOverviewWrapper,
			jwtWrapper,
			policyWrapper,
			accessManagementWrapper,
			applicationsWrapper,
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
		"Sources like: directory, zip file or git URL.",
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
	createScanCmd.PersistentFlags().String(
		commonParams.ScaResolverFlag,
		"",
		"Resolve SCA project dependencies (path to SCA Resolver executable).",
	)
	createScanCmd.PersistentFlags().String(
		commonParams.ScaResolverParamsFlag,
		"",
		fmt.Sprintf("Parameters to use in SCA resolver (requires --%s).", commonParams.ScaResolverFlag),
	)
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
	addResultFormatFlag(
		createScanCmd,
		printer.FormatSummaryConsole,
		printer.FormatJSON,
		printer.FormatSummary,
		printer.FormatSarif,
		printer.FormatSbom,
		printer.FormatPDF,
		printer.FormatSummaryMarkdown,
		printer.FormatGL,
	)
	createScanCmd.PersistentFlags().String(commonParams.APIDocumentationFlag, "", apiDocumentationFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ExploitablePathFlag, "", exploitablePathFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.LastSastScanTime, "", scaLastScanTimeFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ProjecPrivatePackageFlag, "", projectPrivatePackageFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ScaPrivatePackageVersionFlag, "", scaPrivatePackageVersionFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ReportFormatPdfToEmailFlag, "", pdfToEmailFlagDescription)
	createScanCmd.PersistentFlags().String(commonParams.ReportSbomFormatFlag, defaultSbomOption, sbomReportFlagDescription)
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

	createScanCmd.PersistentFlags().String(commonParams.ApplicationName, "", "Name of the application to assign with the project")
	// Link the environment variables to the CLI argument(s).
	err = viper.BindPFlag(commonParams.BranchKey, createScanCmd.PersistentFlags().Lookup(commonParams.BranchFlag))
	if err != nil {
		log.Fatal(err)
	}

	createScanCmd.PersistentFlags().String(commonParams.SSHKeyFlag, "", "Path to ssh private key")

	createScanCmd.PersistentFlags().Int(commonParams.RetrySBOMFlag, commonParams.RetrySBOMDefault, commonParams.RetrySBOMUsage)
	// Temporary flag until SCA supports new api
	createScanCmd.PersistentFlags().Bool(commonParams.ReportSbomFormatLocalFlowFlag, false, "")
	_ = createScanCmd.PersistentFlags().MarkHidden(commonParams.ReportSbomFormatLocalFlowFlag)

	return createScanCmd
}

func findProject(
	applicationID []string,
	projectName string,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
) (string, error) {
	params := make(map[string]string)
	params["names"] = projectName
	resp, _, err := projectsWrapper.Get(params)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(resp.Projects); i++ {
		if resp.Projects[i].Name == projectName {
			return updateProject(resp, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper, projectName, applicationID)
		}
	}
	projectID, err := createProject(projectName, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper, applicationID)
	if err != nil {
		return "", err
	}
	return projectID, nil
}

func createProject(
	projectName string,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationID []string,
) (string, error) {
	projectGroups, _ := cmd.Flags().GetString(commonParams.ProjectGroupList)
	projectTags, _ := cmd.Flags().GetString(commonParams.ProjectTagList)
	projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)
	groupsMap, err := createGroupsMap(projectGroups, groupsWrapper)
	if err != nil {
		return "", err
	}
	var projModel = wrappers.Project{}
	projModel.Name = projectName
	projModel.Groups = getGroupsForRequest(groupsMap)
	projModel.ApplicationIds = applicationID

	if projectPrivatePackage != "" {
		projModel.PrivatePackage, _ = strconv.ParseBool(projectPrivatePackage)
	}
	projModel.Tags = createTagMap(projectTags)
	resp, errorModel, err := projectsWrapper.Create(&projModel)
	projectID := ""
	if errorModel != nil {
		err = errors.Errorf(ErrorCodeFormat, failedCreatingProj, errorModel.Code, errorModel.Message)
	}
	if err == nil {
		projectID = resp.ID
		err = assignGroupsToProject(projectID, projectName, groupsMap, accessManagementWrapper)
	}
	return projectID, err
}

func updateProject(
	resp *wrappers.ProjectsCollectionResponseModel,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	projectName string,
	applicationID []string,

) (string, error) {
	var projectID string
	var projModel = wrappers.Project{}
	projectGroups, _ := cmd.Flags().GetString(commonParams.ProjectGroupList)
	projectTags, _ := cmd.Flags().GetString(commonParams.ProjectTagList)
	projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)
	for i := 0; i < len(resp.Projects); i++ {
		if resp.Projects[i].Name == projectName {
			projectID = resp.Projects[i].ID
		}
		if resp.Projects[i].MainBranch != "" {
			projModel.MainBranch = resp.Projects[i].MainBranch
		}
		if resp.Projects[i].RepoURL != "" {
			projModel.RepoURL = resp.Projects[i].RepoURL
		}
	}
	if projectGroups == "" && projectTags == "" && projectPrivatePackage == "" && len(applicationID) == 0 {
		logger.PrintIfVerbose("No groups, applicationId or tags to update. Skipping project update.")
		return projectID, nil
	}
	if projectPrivatePackage != "" {
		projModel.PrivatePackage, _ = strconv.ParseBool(projectPrivatePackage)
	}
	projModelResp, errModel, err := projectsWrapper.GetByID(projectID)
	if errModel != nil {
		err = errors.Errorf(ErrorCodeFormat, failedGettingProj, errModel.Code, errModel.Message)
	}
	if err != nil {
		return "", err
	}
	projModel.Name = projModelResp.Name
	projModel.Groups = projModelResp.Groups
	projModel.Tags = projModelResp.Tags
	projModel.ApplicationIds = projModelResp.ApplicationIds
	if projectGroups != "" {
		groupsMap, groupErr := createGroupsMap(projectGroups, groupsWrapper)
		if groupErr != nil {
			return "", errors.Errorf("%s: %v", failedUpdatingProj, groupErr)
		}
		logger.PrintIfVerbose("Updating project groups")
		projModel.Groups = getGroupsForRequest(groupsMap)
		err = assignGroupsToProject(projectID, projectName, groupsMap, accessManagementWrapper)
		if err != nil {
			return "", err
		}
	}
	if projectTags != "" {
		logger.PrintIfVerbose("Updating project tags")
		projModel.Tags = createTagMap(projectTags)
	}
	if len(applicationID) > 0 {
		logger.PrintIfVerbose("Updating project applicationIds")
		projModel.ApplicationIds = createApplicationIds(applicationID, projModelResp.ApplicationIds)
	}
	err = projectsWrapper.Update(projectID, &projModel)
	if err != nil {
		return "", errors.Errorf("%s: %v", failedUpdatingProj, err)
	}
	return projectID, nil
}

func createApplicationIds(applicationID, existingApplicationIds []string) []string {
	for _, id := range applicationID {
		if !util.Contains(existingApplicationIds, id) {
			existingApplicationIds = append(existingApplicationIds, id)
		}
	}
	return existingApplicationIds
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
		if len(tag) > 0 {
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

func createTagMap(tagListStr string) map[string]string {
	tagsList := strings.Split(tagListStr, ",")
	tags := make(map[string]string)
	for _, tag := range tagsList {
		if len(tag) > 0 {
			value := ""
			keyValuePair := strings.Split(tag, ":")
			if len(keyValuePair) > 1 {
				value = keyValuePair[1]
			}
			tags[keyValuePair[0]] = value
		}
	}
	return tags
}

func setupScanTypeProjectAndConfig(
	input *[]byte,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	scansWrapper wrappers.ScansWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
) error {
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
		return errors.Errorf("Project name is required")
	}

	applicationName, err := cmd.Flags().GetString(commonParams.ApplicationName)
	if err != nil {
		return err
	}

	var applicationID []string
	if applicationName != "" {
		application, getAppErr := getApplication(applicationName, applicationsWrapper)
		if getAppErr != nil {
			return getAppErr
		}
		if application == nil {
			return errors.Errorf(errorconsts.ApplicationDoesntExistOrNoPermission)
		}
		applicationID = []string{application.ID}
	}

	// We need to convert the project name into an ID
	projectID, findProjectErr := findProject(
		applicationID,
		info["project"].(map[string]interface{})["id"].(string),
		cmd,
		projectsWrapper,
		groupsWrapper,
		accessManagementWrapper,
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
		resubmitConfig, err = getResubmitConfiguration(scansWrapper, projectID, userScanTypes)
		if err != nil {
			return err
		}
	} else if _, ok := info["config"]; !ok {
		err = json.Unmarshal([]byte("[]"), &configArr)
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
	var scaConfig = addScaScan(cmd, resubmitConfig)
	if scaConfig != nil {
		configArr = append(configArr, scaConfig)
	}
	var apiSecConfig = addAPISecScan(cmd)
	if apiSecConfig != nil {
		configArr = append(configArr, apiSecConfig)
	}
	info["config"] = configArr
	*input, err = json.Marshal(info)
	return err
}

func getApplication(applicationName string, applicationsWrapper wrappers.ApplicationsWrapper) (*wrappers.Application, error) {
	if applicationName != "" {
		params := make(map[string]string)
		params["name"] = applicationName
		resp, err := applicationsWrapper.Get(params)
		if err != nil {

			return nil, err
		}
		if resp.Applications != nil && len(resp.Applications) > 0 {
			application := verifyApplicationNameExactMatch(applicationName, resp)

			return application, nil
		}
	}
	return nil, nil
}

func verifyApplicationNameExactMatch(applicationName string, resp *wrappers.ApplicationsResponseModel) *wrappers.Application {
	var application *wrappers.Application
	for i := range resp.Applications {
		if resp.Applications[i].Name == applicationName {
			application = &resp.Applications[i]
			break
		}
	}
	return application
}

func getResubmitConfiguration(scansWrapper wrappers.ScansWrapper, projectID, userScanTypes string) (
	[]wrappers.Config,
	error,
) {
	var allScansModel *wrappers.ScansCollectionResponseModel
	var errorModel *wrappers.ErrorModel
	var err error
	params := make(map[string]string)
	params["project-id"] = projectID
	allScansModel, errorModel, err = scansWrapper.Get(params)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s\n", failedGettingAll)
	}
	// Checking the response for errors
	if errorModel != nil {
		return nil, errors.Errorf(ErrorCodeFormat, failedGettingAll, errorModel.Code, errorModel.Message)
	}
	config := allScansModel.Scans[0].Metadata.Configs
	engines := allScansModel.Scans[0].Engines
	// Check if there are no scan types sent using the flags, and use the latest scan engine types
	if userScanTypes == "" {
		actualScanTypes = strings.Join(engines, ",")
	}
	return config, nil
}

func addSastScan(cmd *cobra.Command, resubmitConfig []wrappers.Config) map[string]interface{} {
	if scanTypeEnabled(commonParams.SastType) {
		sastMapConfig := make(map[string]interface{})
		sastConfig := wrappers.SastConfig{}
		sastMapConfig[resultsMapType] = commonParams.SastType
		incrementalVal, _ := cmd.Flags().GetBool(commonParams.IncrementalSast)
		sastConfig.Incremental = strconv.FormatBool(incrementalVal)
		sastConfig.PresetName, _ = cmd.Flags().GetString(commonParams.PresetName)
		sastConfig.Filter, _ = cmd.Flags().GetString(commonParams.SastFilterFlag)
		for _, config := range resubmitConfig {
			if config.Type != commonParams.SastType {
				continue
			}
			resubmitIncremental := config.Value[configIncremental]
			if resubmitIncremental != nil && !incrementalVal {
				sastConfig.Incremental = resubmitIncremental.(string)
			}
			resubmitPreset := config.Value[configPresetName]
			if resubmitPreset != nil && sastConfig.PresetName == "" {
				sastConfig.PresetName = resubmitPreset.(string)
			}
			resubmitFilter := config.Value[configFilterKey]
			if resubmitFilter != nil && sastConfig.Filter == "" {
				sastConfig.Filter = resubmitFilter.(string)
			}
			resubmitEngineVerbose := config.Value[configEngineVerbose]
			if resubmitEngineVerbose != nil {
				sastConfig.EngineVerbose = resubmitEngineVerbose.(string)
			}
			resubmitLanguageMode := config.Value[configLanguageMode]
			if resubmitLanguageMode != nil {
				sastConfig.LanguageMode = resubmitLanguageMode.(string)
			}
		}
		sastMapConfig[resultsMapValue] = &sastConfig
		return sastMapConfig
	}
	return nil
}

func addKicsScan(cmd *cobra.Command, resubmitConfig []wrappers.Config) map[string]interface{} {
	if scanTypeEnabled(commonParams.KicsType) {
		kicsMapConfig := make(map[string]interface{})
		kicsConfig := wrappers.KicsConfig{}
		kicsMapConfig[resultsMapType] = commonParams.KicsType
		kicsConfig.Filter = deprecatedFlagValue(cmd, commonParams.KicsFilterFlag, commonParams.IacsFilterFlag)
		kicsConfig.Platforms = deprecatedFlagValue(cmd, commonParams.KicsPlatformsFlag, commonParams.IacsPlatformsFlag)
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
			}
		}
		kicsMapConfig[resultsMapValue] = &kicsConfig
		return kicsMapConfig
	}
	return nil
}

func addScaScan(cmd *cobra.Command, resubmitConfig []wrappers.Config) map[string]interface{} {
	if scanTypeEnabled(commonParams.ScaType) {
		scaMapConfig := make(map[string]interface{})
		scaConfig := wrappers.ScaConfig{}
		scaMapConfig[resultsMapType] = commonParams.ScaType
		scaConfig.Filter, _ = cmd.Flags().GetString(commonParams.ScaFilterFlag)
		scaConfig.LastSastScanTime, _ = cmd.Flags().GetString(commonParams.LastSastScanTime)
		scaConfig.PrivatePackageVersion, _ = cmd.Flags().GetString(commonParams.ScaPrivatePackageVersionFlag)
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
		scaMapConfig[resultsMapValue] = &scaConfig
		return scaMapConfig
	}
	return nil
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

func validateScanTypes(cmd *cobra.Command, jwtWrapper wrappers.JWTWrapper) error {
	var scanTypes []string
	allowedEngines, err := jwtWrapper.GetAllowedEngines()
	if err != nil {
		err = errors.Errorf("Error validating scan types: %v", err)
		return err
	}

	userScanTypes, _ := cmd.Flags().GetString(commonParams.ScanTypes)
	if len(userScanTypes) > 0 {
		userScanTypes = strings.ReplaceAll(strings.ToLower(userScanTypes), " ", "")
		userScanTypes = strings.Replace(strings.ToLower(userScanTypes), commonParams.KicsType, commonParams.IacType, 1)

		scanTypes = strings.Split(userScanTypes, ",")
		for _, scanType := range scanTypes {
			if !allowedEngines[scanType] {
				keys := reflect.ValueOf(allowedEngines).MapKeys()
				err = errors.Errorf(engineNotAllowed, scanType, scanType, keys)
				return err
			}
		}
	} else {
		for k := range allowedEngines {
			scanTypes = append(scanTypes, k)
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
	outputFile, err := ioutil.TempFile(os.TempDir(), "cx-*.zip")
	if err != nil {
		return "", errors.Wrapf(err, "Cannot source code temp file.")
	}
	zipWriter := zip.NewWriter(outputFile)
	err = addDirFiles(zipWriter, "", sourceDir, getUserFilters(filter), getIncludeFilters(userIncludeFilter))
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

func getIncludeFilters(userIncludeFilter string) []string {
	return buildFilters(commonParams.BaseFilters, userIncludeFilter)
}

func getUserFilters(filterStr string) []string {
	return buildFilters(nil, filterStr)
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
	files, err := ioutil.ReadDir(parentDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			err = handleDir(zipWriter, baseDir, parentDir, filters, includeFilters, file)
		} else {
			err = handleFile(zipWriter, baseDir, parentDir, filters, includeFilters, file)
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
	// Check if the folder is excluded
	for _, filter := range filters {
		if filter[0] == '!' {
			filterStr := strings.TrimSuffix(filepath.ToSlash(filter[1:]), "/")
			match, err := path.Match(filterStr, file.Name())
			if err != nil {
				return err
			}
			if match {
				logger.PrintIfVerbose("Excluded: " + parentDir + file.Name() + "/")
				return nil
			}
		}
	}
	newParent, newBase := GetNewParentAndBase(parentDir, file, baseDir)
	return addDirFiles(zipWriter, newBase, newParent, filters, includeFilters)
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
	if len(scaResolver) > 0 {
		scaFile, err := ioutil.TempFile("", "sca")
		scaResolverResultsFile = scaFile.Name() + ".json"
		if err != nil {
			return err
		}
		scaResolverParsedParams, err := shlex.Split(scaResolverParams)
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
		args = append(args, scaResolverParsedParams...)
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
	_ = os.Remove(scaResolverResultsFile)
	if err != nil {
		return err
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

func getUploadURLFromSource(cmd *cobra.Command, uploadsWrapper wrappers.UploadsWrapper) (
	url, zipFilePath string,
	err error,
) {
	var preSignedURL string

	sourceDirFilter, _ := cmd.Flags().GetString(commonParams.SourceDirFilterFlag)
	userIncludeFilter, _ := cmd.Flags().GetString(commonParams.IncludeFilterFlag)
	projectName, _ := cmd.Flags().GetString(commonParams.ProjectName)

	zipFilePath, directoryPath, err := definePathForZipFileOrDirectory(cmd)
	if err != nil {
		return "", "", errors.Wrapf(err, "%s: Input in bad format", failedCreating)
	}

	var errorUnzippingFile error

	userProvidedZip := len(zipFilePath) > 0
	unzip := (len(sourceDirFilter) > 0 || len(userIncludeFilter) > 0) && userProvidedZip
	if unzip {
		directoryPath, errorUnzippingFile = UnzipFile(zipFilePath)
		if errorUnzippingFile != nil {
			return "", "", errorUnzippingFile
		}
	}

	if directoryPath != "" {
		var dirPathErr error

		scaResolverParams, scaResolver := getScaResolverFlags(cmd)

		// Make sure scaResolver only runs in sca type of scans
		if strings.Contains(actualScanTypes, commonParams.ScaType) {
			dirPathErr = runScaResolver(directoryPath, scaResolver, scaResolverParams, projectName)
			if dirPathErr != nil {
				if unzip {
					_ = cleanTempUnzipDirectory(directoryPath)
				}
				return "", "", errors.Wrapf(dirPathErr, "ScaResolver error")
			}
		}

		zipFilePath, dirPathErr = compressFolder(directoryPath, sourceDirFilter, userIncludeFilter, scaResolver)
		if unzip {
			dirRemovalErr := cleanTempUnzipDirectory(directoryPath)
			if dirRemovalErr != nil {
				return "", "", dirRemovalErr
			}
		}
		if dirPathErr != nil {
			return "", "", dirPathErr
		}
	}
	if zipFilePath != "" {
		return uploadZip(uploadsWrapper, zipFilePath, unzip, userProvidedZip)
	}
	return preSignedURL, zipFilePath, nil
}

func uploadZip(uploadsWrapper wrappers.UploadsWrapper, zipFilePath string, unzip, userProvidedZip bool) (
	url, zipPath string,
	err error,
) {
	var zipFilePathErr error
	// Send a request to uploads service
	var preSignedURL *string
	preSignedURL, zipFilePathErr = uploadsWrapper.UploadFile(zipFilePath)
	if zipFilePathErr != nil {
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
		if filepath.Ext(sourceTrimmed) == ".zip" {
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
	resultsSbomWrapper wrappers.ResultsSbomWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	jwtWrapper wrappers.JWTWrapper,
	policyWrapper wrappers.PolicyWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := validateScanTypes(cmd, jwtWrapper)
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
		scanModel, zipFilePath, err := createScanModel(
			cmd,
			uploadsWrapper,
			projectsWrapper,
			groupsWrapper,
			scansWrapper,
			accessManagementWrapper,
			applicationsWrapper,
		)
		if err != nil {
			return errors.Errorf("%s", err)
		}
		scanResponseModel, errorModel, err := scansWrapper.Create(scanModel)
		if err != nil {
			return errors.Wrapf(err, "%s", failedCreating)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf(ErrorCodeFormat, failedCreating, errorModel.Code, errorModel.Message)
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
				resultsSbomWrapper,
				resultsPdfReportsWrapper,
				resultsWrapper,
				risksOverviewWrapper)
			if err != nil {
				return err
			}
			// Handling policy response
			policyOverrideFlag, _ := cmd.Flags().GetBool(commonParams.IgnorePolicyFlag)
			if !policyOverrideFlag {
				policyTimeout, _ := cmd.Flags().GetInt(commonParams.PolicyTimeoutFlag)
				if policyTimeout < 0 {
					return errors.Errorf("--%s should be equal or higher than 0", commonParams.PolicyTimeoutFlag)
				}
				policyResponseModel, err = policymanagement.HandlePolicyWait(waitDelay, policyTimeout, policyWrapper, scanResponseModel.ID, scanResponseModel.ProjectID, cmd)
				if err != nil {
					return err
				}
			} else {
				logger.PrintIfVerbose("Skipping policy evaluation")
			}
			err = createReportsAfterScan(cmd, scanResponseModel.ID, scansWrapper, resultsSbomWrapper, resultsPdfReportsWrapper, resultsWrapper, risksOverviewWrapper, policyResponseModel)
			if err != nil {
				return err
			}

			err = applyThreshold(cmd, resultsWrapper, scanResponseModel)
			if err != nil {
				return err
			}
		} else {
			err = createReportsAfterScan(cmd, scanResponseModel.ID, scansWrapper, resultsSbomWrapper, resultsPdfReportsWrapper, resultsWrapper, risksOverviewWrapper, nil)
			if err != nil {
				return err
			}
		}

		cleanUpTempZip(zipFilePath)
		// verify break build from policy
		if policyResponseModel != nil && len(policyResponseModel.Policies) > 0 && policyResponseModel.BreakBuild {
			logger.PrintIfVerbose("Breaking the build due to policy violation")
			return errors.Errorf("Policy Violation - Break Build Enabled. To bypass the policy evaluation and continue with the build, you can use the `--ignore-policy` flag.")
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
) (*wrappers.Scan, string, error) {
	var input = []byte("{}")

	// Define type, project and config in scan model
	err := setupScanTypeProjectAndConfig(&input, cmd, projectsWrapper, groupsWrapper, scansWrapper, applicationsWrapper, accessManagementWrapper)
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
	scanHandler, zipFilePath, err := setupScanHandler(cmd, uploadsWrapper)
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

func setupScanHandler(cmd *cobra.Command, uploadsWrapper wrappers.UploadsWrapper) (
	wrappers.ScanHandler,
	string,
	error,
) {
	zipFilePath := ""
	scanHandler := wrappers.ScanHandler{}
	scanHandler.Branch = viper.GetString(commonParams.BranchKey)

	uploadType := getUploadType(cmd)

	if uploadType == git {
		source, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		scanHandler.RepoURL = strings.TrimSpace(source)
	} else {
		var err error
		var uploadURL string
		uploadURL, zipFilePath, err = getUploadURLFromSource(cmd, uploadsWrapper)
		if err != nil {
			return scanHandler, zipFilePath, err
		}

		scanHandler.UploadURL = uploadURL
	}

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
	resultsSbomWrapper wrappers.ResultsSbomWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
) error {
	err := waitForScanCompletion(
		scanResponseModel,
		waitDelay,
		timeoutMinutes,
		scansWrapper,
		resultsSbomWrapper,
		resultsPdfReportsWrapper,
		resultsWrapper,
		risksOverviewWrapper,
		cmd)
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
	resultsSbomWrapper wrappers.ResultsSbomWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	policyResponseModel *wrappers.PolicyResponseModel,
) error {
	// Create the required reports
	targetFile, _ := cmd.Flags().GetString(commonParams.TargetFlag)
	targetPath, _ := cmd.Flags().GetString(commonParams.TargetPathFlag)
	reportFormats, _ := cmd.Flags().GetString(commonParams.TargetFormatFlag)
	formatPdfToEmail, _ := cmd.Flags().GetString(commonParams.ReportFormatPdfToEmailFlag)
	formatPdfOptions, _ := cmd.Flags().GetString(commonParams.ReportFormatPdfOptionsFlag)
	formatSbomOptions, _ := cmd.Flags().GetString(commonParams.ReportSbomFormatFlag)
	useSCALocalFlow, _ := cmd.Flags().GetBool(commonParams.ReportSbomFormatLocalFlowFlag)
	retrySBOM, _ := cmd.Flags().GetInt(commonParams.RetrySBOMFlag)

	params, err := getFilters(cmd)
	if err != nil {
		return err
	}
	if !strings.Contains(reportFormats, printer.FormatSummaryConsole) {
		reportFormats += "," + printer.FormatSummaryConsole
	}
	scan, errorModel, scanErr := scansWrapper.GetByID(scanID)
	if scanErr != nil {
		return errors.Wrapf(scanErr, "%s", failedGetting)
	}
	if errorModel != nil {
		return errors.Errorf("%s: CODE: %d, %s", failedGettingScan, errorModel.Code, errorModel.Message)
	}
	return CreateScanReport(
		resultsWrapper,
		risksOverviewWrapper,
		resultsSbomWrapper,
		policyResponseModel,
		useSCALocalFlow,
		retrySBOM,
		resultsPdfReportsWrapper,
		scan,
		reportFormats,
		formatPdfToEmail,
		formatPdfOptions,
		formatSbomOptions,
		targetFile,
		targetPath,
		params,
	)
}

func applyThreshold(
	cmd *cobra.Command,
	resultsWrapper wrappers.ResultsWrapper,
	scanResponseModel *wrappers.ScanResponseModel,
) error {
	threshold, _ := cmd.Flags().GetString(commonParams.Threshold)
	if strings.TrimSpace(threshold) == "" {
		return nil
	}

	thresholdMap := parseThreshold(threshold)

	summaryMap, err := getSummaryThresholdMap(resultsWrapper, scanResponseModel)
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
	thresholdMap := make(map[string]int)
	if threshold != "" {
		threshold = strings.ReplaceAll(strings.ReplaceAll(threshold, " ", ""), ",", ";")
		thresholdLimits := strings.Split(strings.ToLower(threshold), ";")
		for _, limits := range thresholdLimits {
			limit := strings.Split(limits, "=")
			engineName := limit[0]
			engineName = strings.Replace(engineName, commonParams.KicsType, commonParams.IacType, 1)
			if len(limit) > 1 {
				intLimit, err := strconv.Atoi(limit[1])
				if err != nil {
					log.Println("Error parsing threshold limit: ", err)
				} else {
					thresholdMap[engineName] = intLimit
				}
			}
		}
	}

	return thresholdMap
}

func getSummaryThresholdMap(resultsWrapper wrappers.ResultsWrapper, scan *wrappers.ScanResponseModel) (
	map[string]int,
	error,
) {
	results, err := ReadResults(resultsWrapper, scan, make(map[string]string))
	if err != nil {
		return nil, err
	}
	summaryMap := make(map[string]int)
	for _, result := range results.Results {
		if isExploitable(result.State) {
			key := strings.ToLower(fmt.Sprintf("%s-%s", strings.Replace(result.Type, commonParams.KicsType, commonParams.IacType, 1), result.Severity))
			summaryMap[key]++
		}
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
	resultsSbomWrapper wrappers.ResultsSbomWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	cmd *cobra.Command,
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
		running, err := isScanRunning(scansWrapper, resultsSbomWrapper, resultsPdfReportsWrapper, resultsWrapper, risksOverviewWrapper, scanResponseModel.ID, cmd)
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
				return errors.Errorf(ErrorCodeFormat, failedCanceling, errorModel.Code, errorModel.Message)
			}
			return errors.Errorf("Timeout of %d minute(s) for scan reached", timeoutMinutes)
		}
		i++
	}
	return nil
}

func isScanRunning(
	scansWrapper wrappers.ScansWrapper,
	resultsSbomWrapper wrappers.ResultsSbomWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	risksOverViewWrapper wrappers.RisksOverviewWrapper,
	scanID string,
	cmd *cobra.Command,
) (bool, error) {
	var scanResponseModel *wrappers.ScanResponseModel
	var errorModel *wrappers.ErrorModel
	var err error
	scanResponseModel, errorModel, err = scansWrapper.GetByID(scanID)
	if err != nil {
		log.Fatal("Cannot source code temp file.", err)
	}
	if errorModel != nil {
		log.Fatalf(fmt.Sprintf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message))
	} else if scanResponseModel != nil {
		if scanResponseModel.Status == wrappers.ScanRunning || scanResponseModel.Status == wrappers.ScanQueued {
			log.Println("Scan status: ", scanResponseModel.Status)
			return true, nil
		}
	}
	log.Println("Scan Finished with status: ", scanResponseModel.Status)
	if scanResponseModel.Status == wrappers.ScanPartial {
		_ = printer.Print(cmd.OutOrStdout(), scanResponseModel.StatusDetails, printer.FormatList)
		reportErr := createReportsAfterScan(
			cmd,
			scanResponseModel.ID,
			scansWrapper,
			resultsSbomWrapper,
			resultsPdfReportsWrapper,
			resultsWrapper,
			risksOverViewWrapper, nil) // check this partial case, how to handle it
		if reportErr != nil {
			return false, errors.New("unable to create report for partial scan")
		}
		return false, errors.New("scan completed partially")
	} else if scanResponseModel.Status != wrappers.ScanCompleted {
		return false, errors.New("scan did not complete successfully")
	}
	return false, nil
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
			return errors.Errorf(ErrorCodeFormat, failedGettingAll, errorModel.Code, errorModel.Message)
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
				return errors.Errorf(ErrorCodeFormat, failedDeleting, errorModel.Code, errorModel.Message)
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
				return errors.Errorf(ErrorCodeFormat, failedCanceling, errorModel.Code, errorModel.Message)
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

	paramsToSast := map[string]string{"scan-ids": strings.Join(scanIDs, ",")}

	sastMetadata, err := sastMetadataWrapper.GetSastMetadataByIDs(paramsToSast)
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
		if strings.Contains(str, v) {
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
		for tries > 0 {
			zipRemoveErr := os.Remove(zipFilePath)
			if zipRemoveErr != nil {
				logger.PrintIfVerbose(
					fmt.Sprintf(
						"Failed to remove temporary zip: %d in %d: %v",
						cleanupMaxRetries-tries,
						cleanupMaxRetries,
						zipRemoveErr,
					),
				)
				tries--
				time.Sleep(time.Duration(cleanupRetryWaitSeconds) * time.Second)
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

func deprecatedFlagValue(cmd *cobra.Command, deprecatedFlagKey, inUseFlagKey string) string {
	flagValue, _ := cmd.Flags().GetString(inUseFlagKey)
	if flagValue == "" {
		flagValue, _ = cmd.Flags().GetString(deprecatedFlagKey)
	}
	return flagValue
}

func validateCreateScanFlags(cmd *cobra.Command) error {
	branch := viper.GetString(commonParams.BranchKey)
	if branch == "" {
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
