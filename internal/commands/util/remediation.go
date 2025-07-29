package util

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/remediation"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	npmPackageFilename        = "package.json"
	permission                = 0644
	permission0666            = 0666
	containerStarting         = "Starting kics container"
	filesContainerLocation    = "/files/"
	filesContainerVolume      = ":/files"
	resultsContainerLocation  = "/kics/"
	containerRemove           = "--rm"
	ContainerImage            = "checkmarx/kics:v2.1.12"
	containerNameFlag         = "--name"
	remediateCommand          = "remediate"
	resultsFlag               = "--results"
	volumeFlag                = "-v"
	containerRun              = "run"
	allRemediationsApplied    = "All remediations available were applied"
	someRemediationsApplied   = "Some remediations available were not applied or there are errors in the results file.Please check kics logs"
	directoryError            = "Failed creating temporary directory for kics remediation command"
	containerWriteFolderError = " Error writing file to temporary directory"
	kicsVerboseFlag           = "-v"
	kicsIncludeIdsFlag        = "--include-ids"
	containerName             = "cli-remediate-kics"
	separator                 = ","
	InvalidEngineError        = "not found in $PATH"
	InvalidEngineErrorWindows = "not found in %PATH%"
	InvalidEngineMessage      = "Please verify if engine is installed"
	NotRunningEngineMessage   = "Please verify if engine is running"
	EngineNoRunningCode       = 125
)

var (
	kicsSimilarityFilter []string
	kicsErrorCodes       = []string{"70"}
)

func NewRemediationCommand() *cobra.Command {
	remediationCmd := &cobra.Command{
		Use:   "remediation",
		Short: "Remediate vulnerabilities",
		Long: `To remediate vulnerabilities for results from a specific engine
	`,
		Example: heredoc.Doc(
			`
			$ cx utils remediation <engine_name> 
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`https://checkmarx.com/resource/documents/en/34965-68653-utils.html
			`,
			),
		},
	}
	scaRemediationCmd := RemediationScaCommand()
	kicsRemediationCmd := RemediationKicsCommand()
	remediationCmd.AddCommand(scaRemediationCmd, kicsRemediationCmd)
	return remediationCmd
}

func RemediationScaCommand() *cobra.Command {
	scaRemediateCmd := &cobra.Command{
		Use:   "sca",
		Short: "Remediate sca vulnerabilities",
		Long: `To remediate package files vulnerabilities detected by the sca engine
	`,
		RunE: runRemediationScaCmd(),
		Example: heredoc.Doc(
			`
			$ cx utils remediation sca --package <package> --package-files <package-files> --package-version <package-version>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`https://checkmarx.com/resource/documents/en/34965-68653-utils.html#UUID-2bd7b006-76ca-b867-e808-185f8f8de34e
			`,
			),
		},
	}
	scaRemediateCmd.PersistentFlags().StringSlice(
		commonParams.RemediationFiles,
		[]string{},
		"Path to input package files to remediate the package version",
	)
	scaRemediateCmd.PersistentFlags().String(commonParams.RemediationPackage, "", "Name of the package to be replaced")
	scaRemediateCmd.PersistentFlags().String(
		commonParams.RemediationPackageVersion,
		"",
		"Version of the package to be replaced",
	)
	_ = scaRemediateCmd.MarkPersistentFlagRequired(commonParams.RemediationFiles)
	_ = scaRemediateCmd.MarkPersistentFlagRequired(commonParams.RemediationPackage)
	_ = scaRemediateCmd.MarkPersistentFlagRequired(commonParams.RemediationPackageVersion)
	return scaRemediateCmd
}

func RemediationKicsCommand() *cobra.Command {
	kicsRemediateCmd := &cobra.Command{
		Use:   "kics",
		Short: "Remediate kics vulnerabilities",
		Long: `To remediate package files vulnerabilities detected by the kics engine
	`,
		RunE: runRemediationKicsCmd(),
		Example: heredoc.Doc(
			`
			$ cx utils remediation kics --results-file <results-file> --kics-files <kics-files>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`https://checkmarx.com/resource/documents/en/34965-68653-utils.html#UUID-e3b49d46-05fc-eacb-ff10-32e89cda0061
			`,
			),
		},
	}
	kicsRemediateCmd.PersistentFlags().String(
		commonParams.KicsRemediationFile,
		"",
		"Path to the kics scan results file. It is used to identify and remediate the kics vulnerabilities",
	)
	kicsRemediateCmd.PersistentFlags().
		StringSliceVar(
			&kicsSimilarityFilter,
			commonParams.KicsSimilarityFilter,
			[]string{},
			"List with the similarity ids that should be remediated : --similarity-ids b42a19486a8e18324a9b2c06147b1c49feb3ba39a0e4aeafec5665e60f98d047,"+
				"9574288c118e8c87eea31b6f0b011295a39ec5e70d83fb70e839b8db4a99eba8",
		)
	kicsRemediateCmd.PersistentFlags().String(
		commonParams.KicsProjectFile,
		"",
		"Absolute path to the folder that contains the file(s) to be remediated",
	)
	kicsRemediateCmd.PersistentFlags().String(
		commonParams.KicsRealtimeEngine,
		"docker",
		"Name in the $PATH for the container engine to run kics. Example:podman.",
	)
	_ = kicsRemediateCmd.MarkPersistentFlagRequired(commonParams.KicsRemediationFile)
	_ = kicsRemediateCmd.MarkPersistentFlagRequired(commonParams.KicsProjectFile)
	return kicsRemediateCmd
}

func runRemediationScaCmd() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Check if input file is supported
		filePaths, _ := cmd.Flags().GetStringSlice(commonParams.RemediationFiles)
		packageName, _ := cmd.Flags().GetString(commonParams.RemediationPackage)
		packageVersion, _ := cmd.Flags().GetString(commonParams.RemediationPackageVersion)
		var err error
		for _, filePath := range filePaths {
			if IsPackageFileSupported(filePath) {
				// read file to string
				fileContent, fileErr := readPackageFile(filePath)
				if fileErr != nil {
					return fileErr
				}
				// Call the parser for each specific package manager
				p := remediation.PackageContentJSON{
					FileContent:       fileContent,
					PackageIdentifier: packageName,
					PackageVersion:    packageVersion,
				}
				parserOutput, fileErr := p.Parser()
				if fileErr != nil {
					return fileErr
				}
				// write to file with replaced package version
				fileErr = writePackageFile(filePath, parserOutput)
				if fileErr != nil {
					return fileErr
				}
			} else {
				logger.Printf("Unsupported package manager file: %s", filePath)
				err = errors.Errorf("Unsupported package manager file")
			}
		}
		return err
	}
}

func readPackageFile(filename string) (string, error) {
	fileBuffer, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(fileBuffer), nil
}

func writePackageFile(filename, output string) error {
	err := os.WriteFile(filename, []byte(output), permission)
	if err != nil {
		return err
	}
	return nil
}

func IsPackageFileSupported(filename string) bool {
	return strings.Contains(filename, npmPackageFilename)
}

func runRemediationKicsCmd() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Create temp location, add it to container volumes and copy the results inside it
		volumeMap, tempDir, err := createKicsRemediateEnv(cmd)
		if err != nil {
			return errors.Errorf("%s", err)
		}
		// Run kics container
		err = runKicsRemediation(cmd, volumeMap, tempDir)
		if err != nil {
			return errors.Errorf("%s", err)
		}
		return nil
	}
}

func runKicsRemediation(cmd *cobra.Command, volumeMap, tempDir string) error {
	kicsFilesPath, _ := cmd.Flags().GetString(commonParams.KicsProjectFile)
	kicsResultsPath, _ := cmd.Flags().GetString(commonParams.KicsRemediationFile)
	_, file := filepath.Split(kicsResultsPath)
	kicsRunArgs := []string{
		containerRun,
		containerRemove,
		volumeFlag,
		volumeMap,
		volumeFlag,
		kicsFilesPath + filesContainerVolume,
		containerNameFlag,
		containerName,
		ContainerImage,
		remediateCommand,
		resultsFlag,
		resultsContainerLocation + file,
		kicsVerboseFlag,
	}
	if len(kicsSimilarityFilter) > 0 {
		kicsRunArgs = append(kicsRunArgs, kicsIncludeIdsFlag)
		kicsSimilarityFilterString := strings.Join(kicsSimilarityFilter, separator)
		kicsRunArgs = append(kicsRunArgs, kicsSimilarityFilterString)
	}
	logger.PrintIfVerbose(containerStarting)
	kicsCmd, _ := cmd.Flags().GetString(commonParams.KicsRealtimeEngine)
	out, err := exec.Command(kicsCmd, kicsRunArgs...).CombinedOutput()
	logger.PrintIfVerbose(string(out))
	/* 	NOTE: the kics container returns 40 instead of 0 when successful!! This
	definitely an incorrect behavior but the following check gets past it.
	*/
	if err == nil {
		logger.PrintIfVerbose(allRemediationsApplied)
		fmt.Println(buildRemediationSummary(string(out)))
		return nil
	}
	if err != nil {
		errorMessage := err.Error()
		extractedErrorCode := errorMessage[strings.LastIndex(errorMessage, " ")+1:]
		os.RemoveAll(tempDir)
		if contains(kicsErrorCodes, extractedErrorCode) {
			logger.PrintIfVerbose(someRemediationsApplied)
			fmt.Println(buildRemediationSummary(string(out)))
			return nil
		}
		exitError, hasExistError := err.(*exec.ExitError)
		if hasExistError {
			if exitError.ExitCode() == EngineNoRunningCode {
				logger.PrintIfVerbose(errorMessage)
				return errors.Errorf(NotRunningEngineMessage)
			}
		} else {
			if strings.Contains(errorMessage, InvalidEngineError) || strings.Contains(errorMessage, InvalidEngineErrorWindows) {
				logger.PrintIfVerbose(errorMessage)
				return errors.Errorf(InvalidEngineMessage)
			}
		}

		return errors.Errorf("Check container engine state. Failed: %s", errorMessage)
	}
	return nil
}

func createKicsRemediateEnv(cmd *cobra.Command) (volume, kicsDir string, err error) {
	kicsDir, err = os.MkdirTemp("", "kics")
	if err != nil {
		return "", "", errors.New(directoryError)
	}
	kicsResultsPath, _ := cmd.Flags().GetString(commonParams.KicsRemediationFile)
	_, file := filepath.Split(kicsResultsPath)
	if file == "" {
		return "", "", errors.New(" No results file was provided")
	}
	kicsFile, err := os.ReadFile(kicsResultsPath)
	if err != nil {
		return "", "", err
	}
	// transform the file_name attribute to match container location
	kicsFile, err = filenameMatcher(kicsFile)
	if err != nil {
		return "", "", err
	}
	destinationFile := fmt.Sprintf("%s/%s", kicsDir, file)
	err = os.WriteFile(destinationFile, kicsFile, permission0666)
	if err != nil {
		return "", "", errors.New(containerWriteFolderError)
	}
	volume = fmt.Sprintf("%s:/kics", kicsDir)
	return volume, kicsDir, nil
}

func filenameMatcher(kicsFile []byte) (kicsFileUpdated []byte, err error) {
	model := wrappers.KicsResultsCollection{}
	err = json.Unmarshal(kicsFile, &model)
	if err != nil {
		return nil, err
	}
	for indexResults := range model.Results {
		for indexLocations := range model.Results[indexResults].Locations {
			file := filepath.Base(model.Results[indexResults].Locations[indexLocations].Filename)
			model.Results[indexResults].Locations[indexLocations].Filename = filesContainerLocation + file
		}
	}
	kicsFileUpdated, err = json.Marshal(model)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return kicsFileUpdated, nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if strings.Contains(str, v) {
			return true
		}
	}
	return false
}

func buildRemediationSummary(out string) (summary string) {
	model := wrappers.KicsRemediationSummary{}
	s := strings.Split(out, "\n")
	// Always comes in the -3 position of the kics output
	availableFixCount := strings.Split(s[len(s)-3], ":")
	if len(availableFixCount) > 0 {
		intAvailableFixCount, _ := strconv.Atoi(strings.ReplaceAll(availableFixCount[1], " ", ""))
		model.AvailableRemediation = intAvailableFixCount
	}
	// Always comes in the -2 position of the kics output
	appliedFixCount := strings.Split(s[len(s)-2], ":")
	if len(appliedFixCount) > 0 {
		intAppliedFixCount, _ := strconv.Atoi(strings.ReplaceAll(appliedFixCount[1], " ", ""))
		model.AppliedRemediation = intAppliedFixCount
	}
	summaryByte, _ := json.Marshal(model)
	return string(summaryByte)
}
