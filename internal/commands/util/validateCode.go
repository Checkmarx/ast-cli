package util

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewValidateCodeCommand() *cobra.Command {
	validateCodeCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate source code for security best practices",
		RunE:  runValidateCodeCmd(),
		Example: heredoc.Doc(
			`
			$ cx utils validate --source-file <sourceFile> 
		`,
		),
	}
	validateCodeCmd.Flags().String(params.ValidateCodeSourceFile, "", "source file to validate")
	validateCodeCmd.Flags().String(params.ValidateCodeResultsFile, "", "validation results file")
	validateCodeCmd.Flags().String(params.ValidateCodeServiceUri, "", "validation service uri")
	validateCodeCmd.Flags().String(params.ValidateCodeServiceKey, "", "validation service key")
	_ = validateCodeCmd.MarkFlagRequired(params.ValidateCodeSourceFile)
	_ = validateCodeCmd.MarkFlagRequired(params.ValidateCodeResultsFile)
	_ = validateCodeCmd.MarkFlagRequired(params.ValidateCodeServiceUri)
	_ = validateCodeCmd.MarkFlagRequired(params.ValidateCodeServiceKey)

	return validateCodeCmd
}

type Payload struct {
	FileName   string `json:"fileName"`
	SourceCode string `json:"sourceCode"`
}

func runValidateCodeCmd() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error
		sourceFileName, _ := cmd.Flags().GetString(params.ValidateCodeSourceFile)
		resultsFileName, _ := cmd.Flags().GetString(params.ValidateCodeResultsFile)
		validateCodeServiceUri, _ := cmd.Flags().GetString(params.ValidateCodeServiceUri)
		validateCodeServiceKey, _ := cmd.Flags().GetString(params.ValidateCodeServiceKey)

		sourceFileContent, err := os.ReadFile(sourceFileName)
		if err != nil {
			return errors.Errorf("Error reading file '%s': %s", sourceFileName, err.Error())
		}
		data := Payload{
			FileName:   sourceFileName,
			SourceCode: string(sourceFileContent),
		}

		payloadBytes, err := json.Marshal(data)
		if err != nil {
			return errors.Errorf("Error marshalling data: %v", err.Error())
		}
		body := bytes.NewReader(payloadBytes)

		req, err := http.NewRequest("POST", validateCodeServiceUri, body)
		if err != nil {
			return errors.Errorf("Error creating request: %v", err.Error())
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+validateCodeServiceKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Errorf("Error making request: %v", err.Error())
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Errorf("Error reading response body: %v", err.Error())
		}

		// write the response to the results file
		err = os.WriteFile(resultsFileName, respBody, 0644)

		return nil
	}
}
