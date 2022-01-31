package commands

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/util"
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
	resolverFilePerm  = 0644
	scaType           = "sca"
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
)

func NewScanCommand(
	scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	logsWrapper wrappers.LogsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage scans",
		Long:  "The scan command enables the ability to manage scans in CxAST.",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/9YuXtw
			`,
			),
		},
	}

	createScanCmd := scanCreateSubCommand(scansWrapper, uploadsWrapper, resultsWrapper, projectsWrapper, groupsWrapper)

	listScansCmd := scanListSubCommand(scansWrapper)

	showScanCmd := scanShowSubCommand(scansWrapper)

	workflowScanCmd := scanWorkflowSubCommand(scansWrapper)

	deleteScanCmd := scanDeleteSubCommand(scansWrapper)

	cancelScanCmd := scanCancelSubCommand(scansWrapper)

	tagsCmd := scanTagsSubCommand(scansWrapper)

	logsCmd := scanLogsSubCommand(logsWrapper)

	addFormatFlagToMultipleCommands(
		[]*cobra.Command{listScansCmd, showScanCmd, workflowScanCmd},
		util.FormatTable, util.FormatList, util.FormatJSON,
	)
	addScanInfoFormatFlag(
		createScanCmd, util.FormatList, util.FormatTable, util.FormatJSON,
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
	)
	return scanCmd
}

func scanLogsSubCommand(logsWrapper wrappers.LogsWrapper) *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Download scan log for selected scan type",
		Long:  "Accepts a scan-id and scan type (sast, kics or sca) and downloads the related scan log",
		Example: heredoc.Doc(
			`
			$ cx scan logs --scan-id <scan Id> --scan-type <sast | sca | kics>
		`,
		),
		RunE: runDownloadLogs(logsWrapper),
	}
	logsCmd.PersistentFlags().String(commonParams.ScanIDFlag, "", "Scan ID to retrieve log for.")
	logsCmd.PersistentFlags().String(commonParams.ScanTypeFlag, "", "Scan type to pull log for, ex: sast, kics or sca.")
	markFlagAsRequired(logsCmd, commonParams.ScanIDFlag)
	markFlagAsRequired(logsCmd, commonParams.ScanTypeFlag)

	return logsCmd
}

func scanTagsSubCommand(scansWrapper wrappers.ScansWrapper) *cobra.Command {
	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "Get a list of all available tags to filter by",
		Long:  "The tags command enables the ability to provide a list of all the available tags in CxAST.",
		Example: heredoc.Doc(
			`
			$ cx scan tags
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/546Xtw
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
		Long:  "The cancel command enables the ability to cancel one or more running scans in CxAST.",
		Example: heredoc.Doc(
			`
			$ cx scan cancel --scan-id <scan ID>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/aY2Xtw
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
				https://checkmarx.atlassian.net/wiki/x/-AuYtw
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
		Long:  "The workflow command enables the ability to provide information about a requested scan workflow in CxAST.",
		Example: heredoc.Doc(
			`
			$ cx scan workflow --scan-id <scan Id>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/Ug2Ytw
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
		Long:  "The show command enables the ability to show information about a requested scan in CxAST.",
		Example: heredoc.Doc(
			`
			$ cx scan show --scan-id <scan Id>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/qAyYtw
			`,
			),
		},
		RunE: runGetScanByIDCommand(scansWrapper),
	}
	addScanIDFlag(showScanCmd, "Scan ID to show.")
	return showScanCmd
}

func scanListSubCommand(scansWrapper wrappers.ScansWrapper) *cobra.Command {
	listScansCmd := &cobra.Command{
		Use:   "list",
		Short: "List all scans in CxAST",
		Long:  "The list command provides a list of all the scans in CxAST.",
		Example: heredoc.Doc(
			`
			$ cx scan list
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/K46Xtw
			`,
			),
		},
		RunE: runListScansCommand(scansWrapper),
	}
	listScansCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterScanListFlagUsage)
	return listScansCmd
}

func scanCreateSubCommand(
	scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
) *cobra.Command {
	createScanCmd := &cobra.Command{
		Use:   "create",
		Short: "Create and run a new scan",
		Long:  "The create command enables the ability to create and run a new scan in CxAST.",
		Example: heredoc.Doc(
			`
			$ cx scan create --project-name <Project Name> -s <path or repository url>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/WguYtw
			`,
			),
		},
		RunE: runCreateScanCommand(scansWrapper, uploadsWrapper, resultsWrapper, projectsWrapper, groupsWrapper),
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
		"Resolve SCA project dependencies",
	)
	createScanCmd.PersistentFlags().String(commonParams.ScanTypes, "", "Scan types, ex: (sast,kics,sca)")
	createScanCmd.PersistentFlags().String(commonParams.TagList, "", "List of tags, ex: (tagA,tagB:val,etc)")
	createScanCmd.PersistentFlags().StringP(
		commonParams.BranchFlag, commonParams.BranchFlagSh,
		commonParams.Branch, commonParams.BranchFlagUsage,
	)
	addResultFormatFlag(createScanCmd, util.FormatSummaryConsole, util.FormatJSON, util.FormatSummary, util.FormatSarif)
	createScanCmd.PersistentFlags().String(commonParams.TargetFlag, "cx_result", "Output file")
	createScanCmd.PersistentFlags().String(commonParams.TargetPathFlag, ".", "Output Path")
	createScanCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterResultsListFlagUsage)
	createScanCmd.PersistentFlags().String(commonParams.ProjectGroupList, "", "List of groups to associate to project")
	createScanCmd.PersistentFlags().String(commonParams.ProjectTagList, "", "List of tags to associate to project")
	createScanCmd.PersistentFlags().String(
		commonParams.Threshold,
		"",
		"Local build threshold. Format <engine>-<severity>=<limit>;<severity>=<limit>",
	)
	// Link the environment variables to the CLI argument(s).
	err = viper.BindPFlag(commonParams.BranchKey, createScanCmd.PersistentFlags().Lookup(commonParams.BranchFlag))
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatal(err)
	}
	return createScanCmd
}

func findProject(
	projectName string,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
) (string, error) {
	params := make(map[string]string)
	params["name"] = projectName
	resp, _, err := projectsWrapper.Get(params)
	if err != nil {
		return "", err
	}
	for i := 0; i < len(resp.Projects); i++ {
		if resp.Projects[i].Name == projectName {
			return resp.Projects[i].ID, nil
		}
	}
	projectID, err := createProject(projectName, cmd, projectsWrapper, groupsWrapper)
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
) (string, error) {
	projectGroups, _ := cmd.Flags().GetString(commonParams.ProjectGroupList)
	projectTags, _ := cmd.Flags().GetString(commonParams.ProjectTagList)
	groupsMap, err := createGroupsMap(projectGroups, groupsWrapper)
	if err != nil {
		return "", err
	}
	var projModel = wrappers.Project{}
	projModel.Name = projectName
	projModel.Groups = groupsMap
	projModel.Tags = createTagMap(projectTags)
	resp, errorModel, err := projectsWrapper.Create(&projModel)
	projectID := ""
	if errorModel != nil {
		err = errors.Errorf(ErrorCodeFormat, failedCreatingProj, errorModel.Code, errorModel.Message)
	}
	if err == nil {
		projectID = resp.ID
	}
	return projectID, err
}

func updateTagValues(input *[]byte, cmd *cobra.Command) {
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

func updateScanRequestValues(
	input *[]byte,
	cmd *cobra.Command,
	sourceType string,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
) error {
	var info map[string]interface{}
	newProjectName, _ := cmd.Flags().GetString(commonParams.ProjectName)
	_ = json.Unmarshal(*input, &info)
	info["type"] = sourceType
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
	// We need to convert the project name into an ID
	projectID, err := findProject(
		info["project"].(map[string]interface{})["id"].(string),
		cmd,
		projectsWrapper,
		groupsWrapper,
	)
	if err != nil {
		return err
	}
	info["project"].(map[string]interface{})["id"] = projectID
	// Handle the scan configuration
	var configArr []interface{}
	if _, ok := info["config"]; !ok {
		err = json.Unmarshal([]byte("[]"), &configArr)
		if err != nil {
			return err
		}
	}
	var sastConfig = addSastScan(cmd)
	if sastConfig != nil {
		configArr = append(configArr, sastConfig)
	}
	var kicsConfig = addKicsScan()
	if kicsConfig != nil {
		configArr = append(configArr, kicsConfig)
	}
	var scaConfig = addScaScan()
	if scaConfig != nil {
		configArr = append(configArr, scaConfig)
	}
	info["config"] = configArr
	*input, err = json.Marshal(info)
	return err
}

func determineScanTypes(cmd *cobra.Command) {
	userScanTypes, _ := cmd.Flags().GetString(commonParams.ScanTypes)
	if len(userScanTypes) > 0 {
		actualScanTypes = userScanTypes
	}
}

func validateScanTypes() {
	scanTypes := strings.Split(actualScanTypes, ",")
	for _, scanType := range scanTypes {
		isValid := false
		if strings.EqualFold(strings.TrimSpace(scanType), "sast") ||
			strings.EqualFold(strings.TrimSpace(scanType), "kics") ||
			strings.EqualFold(strings.TrimSpace(scanType), "sca") {
			isValid = true
		}
		if !isValid {
			log.Println(fmt.Sprintf("Error: unknown scan type: %s", scanType))
			os.Exit(1)
		}
	}
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

func addSastScan(cmd *cobra.Command) map[string]interface{} {
	if scanTypeEnabled("sast") {
		var objArr map[string]interface{}
		_ = json.Unmarshal([]byte("{}"), &objArr)
		newIncremental, _ := cmd.Flags().GetBool(commonParams.IncrementalSast)
		newPresetName, _ := cmd.Flags().GetString(commonParams.PresetName)
		objArr["type"] = "sast"
		var valueMap map[string]interface{}
		_ = json.Unmarshal([]byte("{}"), &valueMap)
		valueMap["incremental"] = fmt.Sprintf("%v", newIncremental)
		if newPresetName == "" {
			newPresetName = "Checkmarx Default"
		}
		valueMap["presetName"] = newPresetName
		objArr["value"] = valueMap
		return objArr
	}
	return nil
}

func addKicsScan() map[string]interface{} {
	if scanTypeEnabled("kics") {
		var objArr map[string]interface{}
		_ = json.Unmarshal([]byte("{}"), &objArr)
		objArr["type"] = "kics"
		var valueMap map[string]interface{}
		_ = json.Unmarshal([]byte("{}"), &valueMap)
		return objArr
	}
	return nil
}

func addScaScan() map[string]interface{} {
	if scanTypeEnabled("sca") {
		var objArr map[string]interface{}
		_ = json.Unmarshal([]byte("{}"), &objArr)
		objArr["type"] = "sca"
		var valueMap map[string]interface{}
		_ = json.Unmarshal([]byte("{}"), &valueMap)
		return objArr
	}
	return nil
}

func compressFolder(sourceDir, filter, userIncludeFilter string, scaResolver string) (string, error) {
	var err error
	scaToolPath := scaResolver
	outputFile, err := ioutil.TempFile(os.TempDir(), "cx-*.zip")
	if err != nil {
		log.Fatal("Cannot source code temp file.", err)
	}
	zipWriter := zip.NewWriter(outputFile)
	err = addDirFiles(zipWriter, "", sourceDir, getUserFilters(filter), getIncludeFilters(userIncludeFilter))
	if err != nil {
		log.Fatal(err)
	}
	if len(scaToolPath) > 0 && len(scaResolverResultsFile) > 0 {
		err = addScaResults(zipWriter)
		if err != nil {
			log.Fatal(err)
		}
	}
	// Close the file
	if err = zipWriter.Close(); err != nil {
		log.Fatal(err)
	}
	stat, err := outputFile.Stat()
	if err != nil {
		log.Fatal(err)
	}
	PrintIfVerbose(fmt.Sprintf("Zip size:  %.2fMB\n", float64(stat.Size())/mbBytes))
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
			PrintIfVerbose("Directory: " + file.Name())
			newParent := parentDir + file.Name() + "/"
			newBase := baseDir + file.Name() + "/"
			err = addDirFilesIgnoreFilter(zipWriter, newBase, newParent)
		} else {
			fileName := parentDir + file.Name()
			PrintIfVerbose("Included: " + fileName)
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
		PrintIfVerbose("Included: " + fileName)
		dat, err := ioutil.ReadFile(parentDir + file.Name())
		if err != nil {
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
		PrintIfVerbose("Excluded: " + fileName)
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
		PrintIfVerbose("The folder " + file.Name() + " is being included")
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
				PrintIfVerbose("Excluded: " + parentDir + file.Name() + "/")
				return nil
			}
		}
	}
	newParent, newBase := GetNewParentAndBase(parentDir, file, baseDir)
	return addDirFiles(zipWriter, newBase, newParent, filters, includeFilters)
}

func GetNewParentAndBase(parentDir string, file fs.FileInfo, baseDir string) (newParent, newBase string) {
	PrintIfVerbose("Directory: " + parentDir + file.Name())
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

func runScaResolver(sourceDir string, scaResolver string) {
	if len(scaResolver) > 0 {
		log.Println("Using SCA resolver: " + scaResolver)
		scaFile, err := ioutil.TempFile("", "sca")
		scaResolverResultsFile = scaFile.Name() + ".json"
		if err != nil {
			log.Fatal(err)
		}
		if scaResolver != "nop" {
			out, err := exec.Command(
				scaResolver,
				"offline",
				"-s",
				sourceDir,
				"-n",
				commonParams.ProjectName,
				"-r",
				scaResolverResultsFile,
			).Output()
			PrintIfVerbose(string(out))
			if err != nil {
				log.Fatal(err)
			}
		} else {
			PrintIfVerbose("Creating 'No Op' resolver file.")
			d1 := []byte("{}")
			err := os.WriteFile(scaResolverResultsFile, d1, resolverFilePerm)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func addScaResults(zipWriter *zip.Writer) error {
	PrintIfVerbose("Included SCA Results: " + ".cxsca-results.json")
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

func determineSourceFile(
	uploadsWrapper wrappers.UploadsWrapper,
	sourcesFile,
	sourceDir,
	sourceDirFilter,
	userIncludeFilter string,
	scaResolver string,
) (string, error) {
	var err error
	var preSignedURL string
	if sourceDir != "" {
		// Make sure scaResolver only runs in sca type of scans
		if strings.Contains(actualScanTypes, scaType) {
			runScaResolver(sourceDir, scaResolver)
		}
		sourcesFile, _ = compressFolder(sourceDir, sourceDirFilter, userIncludeFilter, scaResolver)
	}
	if sourcesFile != "" {
		// Send a request to uploads service
		var preSignedURL *string
		preSignedURL, err = uploadsWrapper.UploadFile(sourcesFile)
		if err != nil {
			return "", errors.Wrapf(err, "%s: Failed to upload sources file\n", failedCreating)
		}
		PrintIfVerbose(fmt.Sprintf("Uploading file to %s\n", *preSignedURL))
		return *preSignedURL, err
	}
	return preSignedURL, err
}

func determineSourceType(sourcesFile string) (zipFile, sourceDir, scanRepoURL string, err error) {
	if strings.HasPrefix(sourcesFile, "https://") ||
		strings.HasPrefix(sourcesFile, "http://") {
		scanRepoURL = sourcesFile

		log.Printf("\n\nScanning branch %s...\n", viper.GetString(commonParams.BranchKey))
	} else {
		info, statErr := os.Stat(sourcesFile)
		if !os.IsNotExist(statErr) {
			if filepath.Ext(sourcesFile) == ".zip" {
				zipFile = sourcesFile
			} else if info != nil && info.IsDir() {
				sourceDir = filepath.ToSlash(sourcesFile)
				if !strings.HasSuffix(sourceDir, "/") {
					sourceDir += "/"
				}
			} else {
				msg := fmt.Sprintf("Sources input has bad format: %v", sourcesFile)
				err = errors.New(msg)
			}
		} else {
			msg := fmt.Sprintf("Sources input has bad format: %v", sourcesFile)
			err = errors.New(msg)
		}
	}
	return zipFile, sourceDir, scanRepoURL, err
}

func runCreateScanCommand(
	scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	resultsWrapper wrappers.ResultsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		branch := viper.GetString(commonParams.BranchKey)
		if branch == "" {
			return errors.Errorf("%s: Please provide a branch", failedCreating)
		}
		scanModel, err := createScanModel(cmd, uploadsWrapper, projectsWrapper, groupsWrapper)
		if err != nil {
			return err
		}
		scanResponseModel, errorModel, err := scansWrapper.Create(scanModel)
		if err != nil {
			return errors.Wrapf(err, "%s", failedCreating)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf(ErrorCodeFormat, failedCreating, errorModel.Code, errorModel.Message)
		} else if scanResponseModel != nil {
			err = printByScanInfoFormat(cmd, toScanView(scanResponseModel))
			if err != nil {
				return errors.Wrapf(err, "%s\n", failedCreating)
			}
		}
		// Wait until the scan is done: Queued, Running
		AsyncFlag, _ := cmd.Flags().GetBool(commonParams.AsyncFlag)
		if !AsyncFlag {
			waitDelay, _ := cmd.Flags().GetInt(commonParams.WaitDelayFlag)
			timeoutMinutes, _ := cmd.Flags().GetInt(commonParams.ScanTimeoutFlag)
			err := handleWait(cmd, scanResponseModel, waitDelay, timeoutMinutes, scansWrapper, resultsWrapper)
			if err != nil {
				return err
			}

			err = applyThreshold(cmd, resultsWrapper, scanResponseModel)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func createScanModel(
	cmd *cobra.Command,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
) (*wrappers.Scan, error) {
	determineScanTypes(cmd)
	validateScanTypes()
	sourceDirFilter, _ := cmd.Flags().GetString(commonParams.SourceDirFilterFlag)
	userIncludeFilter, _ := cmd.Flags().GetString(commonParams.IncludeFilterFlag)
	sourcesFile, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
	sourcesFile, sourceDir, scanRepoURL, err := determineSourceType(strings.TrimSpace(sourcesFile))
	if err != nil {
		return nil, errors.Wrapf(err, "%s: Input in bad format", failedCreating)
	}
	uploadType := getUploadType(sourceDir, sourcesFile)
	var input = []byte("{}")
	err = updateScanRequestValues(&input, cmd, uploadType, projectsWrapper, groupsWrapper)
	if err != nil {
		return nil, err
	}
	updateTagValues(&input, cmd)
	scanModel := wrappers.Scan{}
	// Try to parse to a scan model in order to manipulate the request payload
	err = json.Unmarshal(input, &scanModel)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: Input in bad format", failedCreating)
	}
	// Get sca resolver flag
	scaResolver, errs := cmd.Flags().GetString(commonParams.ScaResolverFlag)
	if errs != nil {
		scaResolver = ""
	}
	// Setup the project handler (either git or upload)
	pHandler, err := setupProjectHandler(
		uploadsWrapper,
		sourcesFile,
		sourceDir,
		sourceDirFilter,
		userIncludeFilter,
		scanRepoURL,
		scaResolver,
	)
	scanModel.Handler, _ = json.Marshal(pHandler)
	if err != nil {
		return nil, err
	}
	return &scanModel, nil
}

func getUploadType(sourceDir, sourcesFile string) string {
	if sourceDir != "" || sourcesFile != "" {
		return "upload"
	}
	return "git"
}

func setupProjectHandler(
	uploadsWrapper wrappers.UploadsWrapper,
	sourcesFile,
	sourceDir,
	sourceDirFilter,
	userIncludeFilter,
	scanRepoURL string,
	scaResolver string,

) (wrappers.UploadProjectHandler, error) {
	pHandler := wrappers.UploadProjectHandler{}
	pHandler.Branch = viper.GetString(commonParams.BranchKey)
	var err error
	pHandler.UploadURL, err = determineSourceFile(
		uploadsWrapper,
		sourcesFile,
		sourceDir,
		sourceDirFilter,
		userIncludeFilter,
		scaResolver,
	)
	pHandler.RepoURL = scanRepoURL
	return pHandler, err
}

func handleWait(
	cmd *cobra.Command,
	scanResponseModel *wrappers.ScanResponseModel,
	waitDelay,
	timeoutMinutes int,
	scansWrapper wrappers.ScansWrapper,
	resultsWrapper wrappers.ResultsWrapper,
) error {
	err := waitForScanCompletion(scanResponseModel, waitDelay, timeoutMinutes, scansWrapper)
	if err != nil {
		verboseFlag, _ := cmd.Flags().GetBool(commonParams.DebugFlag)
		if verboseFlag {
			log.Println("Printing workflow logs")
			taskResponseModel, _, _ := scansWrapper.GetWorkflowByID(scanResponseModel.ID)
			_ = util.Print(cmd.OutOrStdout(), taskResponseModel, util.FormatList)
		}
		return err
	}

	// Create the required reports
	targetFile, _ := cmd.Flags().GetString(commonParams.TargetFlag)
	targetPath, _ := cmd.Flags().GetString(commonParams.TargetPathFlag)
	reportFormats, _ := cmd.Flags().GetString(commonParams.TargetFormatFlag)
	params, err := getFilters(cmd)
	if err != nil {
		return err
	}
	if !strings.Contains(reportFormats, util.FormatSummaryConsole) {
		reportFormats += "," + util.FormatSummaryConsole
	}
	return CreateScanReport(
		resultsWrapper,
		scansWrapper,
		scanResponseModel.ID,
		reportFormats,
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

	summaryMap, err := getSummaryThresholdMap(resultsWrapper, scanResponseModel.ID)
	if err != nil {
		return err
	}

	var errorBuilder strings.Builder
	var messageBuilder strings.Builder
	for key, thresholdLimit := range thresholdMap {
		currentValue := summaryMap[key]
		failed := currentValue >= thresholdLimit
		logMessage := fmt.Sprintf(thresholdLog, key, thresholdLimit, currentValue)
		PrintIfVerbose(logMessage)

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
		thresholdLimits := strings.Split(threshold, ";")
		for _, limits := range thresholdLimits {
			limit := strings.Split(limits, "=")
			if len(limit) > 1 {
				thresholdMap[strings.ToLower(limit[0])], _ = strconv.Atoi(limit[1])
			}
		}
	}

	return thresholdMap
}

func getSummaryThresholdMap(resultsWrapper wrappers.ResultsWrapper, scanID string) (map[string]int, error) {
	results, err := ReadResults(resultsWrapper, scanID, make(map[string]string))
	if err != nil {
		return nil, err
	}
	summaryMap := make(map[string]int)
	for _, result := range results.Results {
		key := strings.ToLower(fmt.Sprintf("%s-%s", result.Type, result.Severity))
		summaryMap[key]++
	}
	return summaryMap, nil
}

func waitForScanCompletion(
	scanResponseModel *wrappers.ScanResponseModel,
	waitDelay,
	timeoutMinutes int,
	scansWrapper wrappers.ScansWrapper,
) error {
	log.Println("Wait for scan to complete", scanResponseModel.ID, scanResponseModel.Status)
	timeout := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)
	time.Sleep(time.Duration(waitDelay) * time.Second)
	for {
		running, err := isScanRunning(scansWrapper, scanResponseModel.ID)
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
		time.Sleep(time.Duration(waitDelay) * time.Second)
	}
	return nil
}

func isScanRunning(scansWrapper wrappers.ScansWrapper, scanID string) (bool, error) {
	var scanResponseModel *wrappers.ScanResponseModel
	var errorModel *wrappers.ErrorModel
	var err error
	scanResponseModel, errorModel, err = scansWrapper.GetByID(scanID)
	if err != nil {
		log.Fatal("Cannot source code temp file.", err)
	}
	if errorModel != nil {
		log.Fatal(fmt.Sprintf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message))
	} else if scanResponseModel != nil {
		if scanResponseModel.Status == "Running" || scanResponseModel.Status == "Queued" {
			log.Println("Scan status: ", scanResponseModel.Status)
			return true, nil
		}
	}
	log.Println("Scan Finished with status: ", scanResponseModel.Status)
	if scanResponseModel.Status != "Completed" {
		return false, errors.New("scan did not complete successfully")
	}
	return false, nil
}

func runListScansCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
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
			err = printByFormat(cmd, toScanViews(allScansModel.Scans))
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
		logText, err := logsWrapper.GetLog(scanID, scanType)
		if err != nil {
			return err
		}
		fmt.Print(logText)
		return nil
	}
}

type scanView struct {
	ID        string `format:"name:Scan ID"`
	ProjectID string `format:"name:Project ID"`
	Status    string
	CreatedAt time.Time `format:"name:Created at;time:01-02-06 15:04:05"`
	UpdatedAt time.Time `format:"name:Updated at;time:01-02-06 15:04:05"`
	Branch    string
	Tags      map[string]string
	Initiator string
	Origin    string
}

func toScanViews(scans []wrappers.ScanResponseModel) []*scanView {
	views := make([]*scanView, len(scans))
	for i := 0; i < len(scans); i++ {
		views[i] = toScanView(&scans[i])
	}
	return views
}

func toScanView(scan *wrappers.ScanResponseModel) *scanView {
	var origin string
	if scan.UserAgent != "" {
		ua := user_agent.New(scan.UserAgent)
		name, version := ua.Browser()
		origin = name + " " + version
	}

	return &scanView{
		ID:        scan.ID,
		Status:    string(scan.Status),
		CreatedAt: scan.CreatedAt,
		UpdatedAt: scan.UpdatedAt,
		ProjectID: scan.ProjectID,
		Branch:    scan.Branch,
		Tags:      scan.Tags,
		Initiator: scan.Initiator,
		Origin:    origin,
	}
}
