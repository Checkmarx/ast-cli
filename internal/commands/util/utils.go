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
	gitURLRegex = "(?P<G1>:git|ssh|https?|git@[-\\w.]+):(\\/\\/)?(?P<G2>.*?)(\\.git)?$"
	sshURLRegex = "^(?P<user>.*?)@(?P<host>.*?):(?:(?P<port>.*?)/)?(?P<path>.*?/.*?)$"
	invalidFlag = "Value of %s is invalid"
	mbBytes     = 1024.0 * 1024.0
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

func CompressFile(sourceFilePath, targetFileName string) (string, error) {
	outputFile, err := os.CreateTemp(os.TempDir(), "cx-*.zip")
	if err != nil {
		return "", errors.Wrapf(err, "Cannot create temp file")
	}
	defer func(outputFile *os.File) {
		CloseFileErr := outputFile.Close()
		if CloseFileErr != nil {
			logger.PrintfIfVerbose("Failed to close file: %s", outputFile.Name())
		}
	}(outputFile)

	zipWriter := zip.NewWriter(outputFile)
	defer func(zipWriter *zip.Writer) {
		closeZipWriterError := zipWriter.Close()
		if closeZipWriterError != nil {
			logger.PrintfIfVerbose("Failed to close zip writer: %s", outputFile.Name())
		}
	}(zipWriter)

	dataFile, err := os.Open(sourceFilePath)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to open file: %s", sourceFilePath)
	}
	defer func(dataFile *os.File) {
		closeDataFileError := dataFile.Close()
		if closeDataFileError != nil {
			logger.PrintfIfVerbose("Failed to close file: %s", dataFile.Name())
		}
	}(dataFile)

	folderNameBeginsIndex := strings.Index(outputFile.Name(), "cx-")
	if folderNameBeginsIndex == -1 {
		return "", errors.Errorf("Failed to find folder name in output file name")
	}
	folderName := outputFile.Name()[folderNameBeginsIndex:]
	folderName = strings.TrimSuffix(folderName, ".zip")

	f, err := zipWriter.Create(filepath.Join(folderName, targetFileName))
	if err != nil {
		return "", err
	}

	_, err = io.Copy(f, dataFile)
	if err != nil {
		return "", err
	}

	stat, err := outputFile.Stat()
	if err != nil {
		return "", err
	}

	fmt.Printf("Zip size: %.3fMB\n", float64(stat.Size())/mbBytes)
	return outputFile.Name(), nil
}
