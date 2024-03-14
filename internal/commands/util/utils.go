package util

import (
	"archive/zip"
	"fmt"
	"io"
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
	gitURLRegex     = "(?P<G1>:git|ssh|https?|git@[-\\w.]+):(\\/\\/)?(?P<G2>.*?)(\\.git)?$"
	sshURLRegex     = "^(?P<user>.*?)@(?P<host>.*?):(?:(?P<port>.*?)/)?(?P<path>.*?/.*?)$"
	invalidFlag     = "Value of %s is invalid"
	mbBytes         = 1024.0 * 1024.0
	directoryPrefix = "cx-"
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
	envCheckCmd := NewEnvCheckCommand()

	completionCmd := NewCompletionCommand()

	prDecorationCmd := NewPRDecorationCommand(prWrapper, policyWrapper, scansWrapper)

	remediationCmd := NewRemediationCommand()

	learnMoreCmd := NewLearnMoreCommand(learnMoreWrapper)

	tenantCmd := NewTenantConfigurationCommand(tenantWrapper)

	maskSecretsCmd := NewMaskSecretsCommand(chatWrapper)

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
func Contains(array []string, val string) bool {
	for _, e := range array {
		if e == val {
			return true
		}
	}
	return false
}

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

func CompressFile(sourceFilePath, targetFileName string, createdDirectoryPrefix ...string) (string, error) {
	if len(createdDirectoryPrefix) == 0 {
		createdDirectoryPrefix = append(createdDirectoryPrefix, directoryPrefix)
	}
	outputFile, err := os.CreateTemp(os.TempDir(), createdDirectoryPrefix[0]+"*.zip")
	if err != nil {
		return "", errors.Wrapf(err, "Cannot create temp file")
	}

	zipWriter := zip.NewWriter(outputFile)

	dataFile, err := os.Open(sourceFilePath)
	if err != nil {
		logger.PrintfIfVerbose("Failed to open file: %s", sourceFilePath)
	}

	folderNameBeginsIndex := strings.Index(outputFile.Name(), "cx-")
	if folderNameBeginsIndex == -1 {
		logger.PrintfIfVerbose("Failed to find folder name in file: %s", outputFile.Name())
	}
	folderName := outputFile.Name()[folderNameBeginsIndex:]
	folderName = strings.TrimSuffix(folderName, ".zip")

	f, err := zipWriter.Create(filepath.Join(folderName, targetFileName))
	if err != nil {
		logger.PrintfIfVerbose("Failed to create file in zip: %s", targetFileName)
	}

	_, err = io.Copy(f, dataFile)
	if err != nil {
		logger.PrintfIfVerbose("Failed to copy file to zip: %s", targetFileName)
	}

	stat, err := outputFile.Stat()
	if err != nil {
		logger.PrintfIfVerbose("Failed to get file stat: %s", outputFile.Name())
	} else {
		fmt.Printf("Zip size: %.3fMB\n", float64(stat.Size())/mbBytes)
	}

	CloseFilesAndWriter(zipWriter, dataFile, outputFile)
	return outputFile.Name(), nil
}

func CloseFilesAndWriter(writer *zip.Writer, files ...*os.File) {
	for _, file := range files {
		if file != nil {
			err := file.Close()
			if err != nil {
				logger.PrintfIfVerbose("Failed to close file: %s", file.Name())
			}
		}
	}
	if writer != nil {
		err := writer.Close()
		if err != nil {
			logger.PrintfIfVerbose("Failed to close zip writer")
		}
	}
}
