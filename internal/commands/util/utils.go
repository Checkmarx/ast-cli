package util

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/usercount"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	gitURLRegex            = "(?P<G1>:git|ssh|https?|git@[-\\w.]+):(\\/\\/)?(?P<G2>.*?)(\\.git)?$"
	sshURLRegex            = "^(?P<user>.*?)@(?P<host>.*?):(?:(?P<port>.*?)/)?(?P<path>.*?/.*?)$"
	invalidFlag            = "Value of %s is invalid"
	mbBytes                = 1024.0 * 1024.0
	defaultDirectoryPrefix = "cx-"
)

func NewUtilsCommand(
	gitHubWrapper wrappers.GitHubWrapper,
	azureWrapper wrappers.AzureWrapper,
	bitBucketWrapper wrappers.BitBucketWrapper,
	bitBucketServerWrapper bitbucketserver.Wrapper,
	gitLabWrapper wrappers.GitLabWrapper,
	prWrapper wrappers.PRWrapper,
	learnMoreWrapper wrappers.LearnMoreWrapper,
	tenantWrapper wrappers.TenantConfigurationWrapper,
	chatWrapper wrappers.ChatWrapper,
	policyWrapper wrappers.PolicyWrapper,
	scansWrapper wrappers.ScansWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
	byorWrapper wrappers.ByorWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) *cobra.Command {
	utilsCmd := &cobra.Command{
		Use:   "utils",
		Short: "Utility functions",
		Long:  "The utils command enables the ability to perform Checkmarx One utility functions.",
		Example: heredoc.Doc(
			`
			$ cx utils env
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68653-utils.html
			`,
			),
		},
	}

	importCmd := NewImportCommand(projectsWrapper, uploadsWrapper, groupsWrapper, accessManagementWrapper, byorWrapper, applicationsWrapper, featureFlagsWrapper)

	envCheckCmd := NewEnvCheckCommand()

	completionCmd := NewCompletionCommand()

	prDecorationCmd := NewPRDecorationCommand(prWrapper, policyWrapper, scansWrapper)

	remediationCmd := NewRemediationCommand()

	learnMoreCmd := NewLearnMoreCommand(learnMoreWrapper)

	tenantCmd := NewTenantConfigurationCommand(tenantWrapper)

	maskSecretsCmd := NewMaskSecretsCommand(chatWrapper)

	// flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, featureFlagsConstants.ByorEnabled)
	// if flagResponse.Status {
	utilsCmd.AddCommand(importCmd)
	//}

	utilsCmd.AddCommand(
		completionCmd,
		envCheckCmd,
		learnMoreCmd,
		usercount.NewUserCountCommand(
			gitHubWrapper,
			azureWrapper,
			bitBucketWrapper,
			bitBucketServerWrapper,
			gitLabWrapper,
		),
		prDecorationCmd,
		remediationCmd,
		tenantCmd,
		maskSecretsCmd,
	)

	return utilsCmd
}

// Contains Tests if a string exists in the provided array/**

func executeTestCommand(cmd *cobra.Command, args ...string) error {
	fmt.Println("Executing command with args ", args)
	cmd.SetArgs(args)
	cmd.SilenceUsage = false
	return cmd.Execute()
}

// IsGitURL Check if provided URL is a valid git URL (http or ssh)
func IsGitURL(url string) bool {
	compiledRegex := regexp.MustCompile(gitURLRegex)
	urlParts := compiledRegex.FindStringSubmatch(url)

	if urlParts == nil || len(urlParts) < 4 {
		return false
	}

	return len(urlParts[1]) > 0 && len(urlParts[3]) > 0
}

// IsSSHURL Check if provided URL is a valid ssh URL
func IsSSHURL(url string) bool {
	isGitURL, _ := regexp.MatchString(sshURLRegex, url)

	return isGitURL
}

// ReadFileAsString Read a file and return its content as string
func ReadFileAsString(path string) (string, error) {
	content, err := os.ReadFile(path)

	if err != nil {
		return "", err
	}

	return string(content), nil
}

// IsDirOrSymLinkToDir Check if provided DirEntry is a directory or symbolic link to a directory
func IsDirOrSymLinkToDir(parentDir string, fileInfo fs.FileInfo) bool {
	if fileInfo.IsDir() {
		return true
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		symlinkPath := filepath.Join(parentDir, fileInfo.Name())
		realPath, err := os.Readlink(symlinkPath)
		if err != nil {
			fmt.Println("Error reading symlink:", err)
			return false
		}

		targetInfo, err := os.Stat(realPath)
		if err != nil {
			fmt.Println("Error getting target info:", err)
			return false
		}
		return targetInfo.IsDir()
	}

	return false
}

func CompressFile(sourceFilePath, targetFileName string, createdDirectoryPrefix ...string) (string, error) {
	if len(createdDirectoryPrefix) == 0 || createdDirectoryPrefix[0] == "" {
		createdDirectoryPrefix = []string{defaultDirectoryPrefix}
	}

	outputFile, err := os.CreateTemp(os.TempDir(), createdDirectoryPrefix[0]+"*.zip")

	if err != nil {
		return "", errors.Wrapf(err, "Cannot create temp file")
	}
	defer CloseOutputFile(outputFile)

	zipWriter := zip.NewWriter(outputFile)
	defer CloseZipWriter(zipWriter, outputFile)

	folderName, nameERR := extractFolderNameFromZipPath(outputFile.Name(), createdDirectoryPrefix[0])
	if nameERR != nil {
		return "", nameERR
	}

	dataFile, readErr := os.ReadFile(sourceFilePath)
	if readErr != nil {
		logger.PrintfIfVerbose("Failed to read file: %s", sourceFilePath)
	}

	f, err := zipWriter.Create(filepath.Join(folderName, targetFileName))
	if err != nil {
		logger.PrintfIfVerbose("Failed to create file in zip: %s", targetFileName)
	}
	_, err = f.Write(dataFile)
	if err != nil {
		logger.PrintfIfVerbose("Failed to write file to zip: %s", targetFileName)
	}
	return outputFile.Name(), nil
}

func extractFolderNameFromZipPath(outputFileName, dirPrefix string) (string, error) {
	folderNameBeginsIndex := strings.Index(outputFileName, dirPrefix)
	if folderNameBeginsIndex == -1 {
		return "", errors.New("Failed to extract folder name from zip path: " + outputFileName + " with prefix: " + dirPrefix)
	}
	return strings.TrimSuffix(outputFileName[folderNameBeginsIndex:], ".zip"), nil
}

func CloseOutputFile(outputFile *os.File) {
	stat, statErr := outputFile.Stat()
	CloseFileErr := outputFile.Close()
	if CloseFileErr != nil {
		logger.PrintfIfVerbose("Failed to close file: %s", outputFile.Name())
	}
	if statErr != nil {
		logger.PrintfIfVerbose("Failed to get file stat: %s", outputFile.Name())
	}
	if stat != nil {
		fmt.Printf("Zip size: %.3fMB\n", float64(stat.Size())/mbBytes)
	}
}

func CloseZipWriter(zipWriter *zip.Writer, outputFile *os.File) {
	closeZipWriterError := zipWriter.Close()
	if closeZipWriterError != nil {
		logger.PrintfIfVerbose("Failed to close zip writer: %s", outputFile.Name())
	}
}
