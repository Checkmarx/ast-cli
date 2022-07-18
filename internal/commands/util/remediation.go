package util

import (
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/remediation"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	npmPackageFilename = "package.json"
	permission         = 0644
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
				`	
			`,
			),
		},
	}
	scaRemediationCmd := RemediationScaCommand()
	remediationCmd.AddCommand(scaRemediationCmd)
	return remediationCmd
}

func RemediationScaCommand() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "sca",
		Short: "Remediate sca vulnerabilities",
		Long: `To remediate package files vulnerabilities detected by the sca engine
	`,
		RunE: runRemediationCmd(),
		Example: heredoc.Doc(
			`
			$ cx utils remediation sca --package <package> --package-file <package-file> --package-version <package-version>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`	
			`,
			),
		},
	}
	completionCmd.PersistentFlags().String(commonParams.RemediationFile, "", "Path to input package file to remediate the package version")
	completionCmd.PersistentFlags().String(commonParams.RemediationPackage, "", "Name of the package to be replaced")
	completionCmd.PersistentFlags().String(commonParams.RemediationPackageVersion, "", "Version of the package to be replaced")
	_ = completionCmd.MarkPersistentFlagRequired(commonParams.RemediationFile)
	_ = completionCmd.MarkPersistentFlagRequired(commonParams.RemediationPackage)
	_ = completionCmd.MarkPersistentFlagRequired(commonParams.RemediationPackageVersion)
	return completionCmd
}

func runRemediationCmd() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Check if input file is supported
		filePath, _ := cmd.Flags().GetString(commonParams.RemediationFile)
		packageName, _ := cmd.Flags().GetString(commonParams.RemediationPackage)
		packageVersion, _ := cmd.Flags().GetString(commonParams.RemediationPackageVersion)
		if isPackageFileSupported(filePath) {
			// read file to string
			fileContent, err := readPackageFile(filePath)
			if err != nil {
				return err
			}
			// Call the parser for each specific package manager
			p := remediation.PackageContentJSON{FileContent: fileContent, PackageIdentifier: packageName, PackageVersion: packageVersion}
			parserOutput, err := p.Parser()
			if err != nil {
				return err
			}
			// write to file with replaced package version
			err = writePackageFile(filePath, parserOutput)
			if err != nil {
				return err
			}
		} else {
			return errors.Errorf("Unsupported package manager file")
		}
		return nil
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

func isPackageFileSupported(filename string) bool {
	var r = false
	if strings.Contains(filename, npmPackageFilename) {
		r = true
	}
	return r
}
