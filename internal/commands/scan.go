package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"
	wrappers "github.com/checkmarxDev/ast-cli/internal/wrappers"
	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/v1/rest"

	"github.com/spf13/cobra"
)

const (
	failedCreating    = "Failed creating a scan"
	failedGetting     = "Failed showing a scan"
	failedGettingTags = "Failed getting tags"
	failedDeleting    = "Failed deleting a scan"
	failedGettingAll  = "Failed listing"
)

var (
	filterScanListFlagUsage = fmt.Sprintf("Filter the list of scans. Available filters are: %s",
		strings.Join([]string{
			commonParams.ScanIDsQueryParam,
			commonParams.LimitQueryParam,
			commonParams.OffsetQueryParam,
			commonParams.FromDateQueryParam,
			commonParams.ToDateQueryParam,
			commonParams.StatusQueryParam,
			commonParams.TagsQueryParam,
			commonParams.ProjectIDQueryParam}, ","))
)

func NewScanCommand(scansWrapper wrappers.ScansWrapper, uploadsWrapper wrappers.UploadsWrapper) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage scans",
	}

	createScanCmd := &cobra.Command{
		Use:   "create",
		Short: "Create and run a new scan",
		RunE:  runCreateScanCommand(scansWrapper, uploadsWrapper),
	}
	createScanCmd.PersistentFlags().StringP(sourcesFlag, sourcesFlagSh, "",
		"A path to the sources file to scan")
	createScanCmd.PersistentFlags().StringP(inputFlag, inputFlagSh, "",
		"The object representing the requested scan, in JSON format")
	createScanCmd.PersistentFlags().StringP(inputFileFlag, inputFileFlagSh, "",
		"A file holding the requested scan object in JSON format. Takes precedence over --input")
	createScanCmd.PersistentFlags().StringSliceP(scanTagsFlag, scanTagsFlagSh, []string{},
		"Scan tags")

	listScansCmd := &cobra.Command{
		Use:   "list",
		Short: "List all scans in the system",
		RunE:  runListScansCommand(scansWrapper),
	}
	listScansCmd.PersistentFlags().StringSlice(filterFlag, []string{}, filterScanListFlagUsage)

	showScanCmd := &cobra.Command{
		Use:   "show <scan id>",
		Short: "Show information about a scan",
		RunE:  runGetScanByIDCommand(scansWrapper),
	}

	deleteScanCmd := &cobra.Command{
		Use:   "delete <scan id>",
		Short: "Stops a scan from running",
		RunE:  runDeleteScanCommand(scansWrapper),
	}

	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "Get a list of all available tags to filter by",
		RunE:  runGetTagsCommand(scansWrapper),
	}

	scanCmd.AddCommand(createScanCmd, showScanCmd, listScansCmd, deleteScanCmd, tagsCmd)
	return scanCmd
}

func runCreateScanCommand(scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var input []byte
		var err error

		var scanInputFile string
		var scanInput string
		var sourcesFile string

		scanInput, _ = cmd.Flags().GetString(inputFlag)
		scanInputFile, _ = cmd.Flags().GetString(inputFileFlag)
		sourcesFile, _ = cmd.Flags().GetString(sourcesFlag)
		scanTagsRaw, _ := cmd.Flags().GetStringSlice(scanTagsFlag)

		PrintIfVerbose(fmt.Sprintf("%s: %s", inputFlag, scanInput))
		PrintIfVerbose(fmt.Sprintf("%s: %s", inputFileFlag, scanInputFile))
		PrintIfVerbose(fmt.Sprintf("%s: %s", sourcesFlag, sourcesFile))
		PrintIfVerbose(fmt.Sprintf("%s: %v", scanTagsFlag, scanTagsRaw))

		if scanInputFile != "" {
			// Reading from input file
			PrintIfVerbose(fmt.Sprintf("Reading input from file %s", scanInputFile))
			input, err = ioutil.ReadFile(scanInputFile)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to open input file", failedCreating)
			}
			wd, _ := os.Getwd()
			PrintIfVerbose(fmt.Sprintf("Input file full path is  %s", filepath.Join(wd, scanInputFile)))
		} else if scanInput != "" {
			// Reading from standard input
			PrintIfVerbose("Reading input from console")
			input = bytes.NewBufferString(scanInput).Bytes()
		} else {
			// No input was given
			return errors.Errorf("%s: no input was given\n", failedCreating)
		}
		var scanModel = scansRESTApi.Scan{}
		var scanResponseModel *scansRESTApi.ScanResponseModel
		var errorModel *scansRESTApi.ErrorModel
		// Try to parse to a scan model in order to manipulate the request payload
		err = json.Unmarshal(input, &scanModel)
		if err != nil {
			return errors.Wrapf(err, "%s: Input in bad format", failedCreating)
		}
		if sourcesFile != "" {
			// Send a request to uploads service
			var preSignedURL *string
			preSignedURL, err = uploadsWrapper.UploadFile(sourcesFile)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to upload sources file\n", failedCreating)
			}
			PrintIfVerbose(fmt.Sprintf("Uploading file to %s\n", *preSignedURL))

			// We are in upload mode - populate fields accordingly
			projectHandlerModel := scansRESTApi.UploadProjectHandler{
				URL: *preSignedURL,
			}
			var projectHandlerModelSerialized []byte
			projectHandlerModelSerialized, err = json.Marshal(projectHandlerModel)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to upload sources file: Failed to serialize project handler",
					failedCreating)
			}
			scanModel.Project.Type = scansRESTApi.UploadProject
			scanModel.Project.Handler = projectHandlerModelSerialized
		}

		if len(scanTagsFlag) > 0 {
			tags, perr := parseTags(scanTagsRaw)
			if perr != nil {
				return errors.Wrapf(err, "%s: Tags in bad format", failedCreating)
			}
			scanModel.Tags = tags
		}
		var payload []byte
		payload, _ = json.Marshal(scanModel)
		PrintIfVerbose(fmt.Sprintf("Payload to scans service: %s\n", string(payload)))

		scanResponseModel, errorModel, err = scansWrapper.Create(&scanModel)
		if err != nil {
			return errors.Wrapf(err, "%s", failedCreating)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedCreating, errorModel.Code, errorModel.Message)
		} else if scanResponseModel != nil {
			err = Print(cmd.OutOrStdout(), toScanView(scanResponseModel))
			if err != nil {
				return errors.Wrapf(err, "%s\n", failedCreating)
			}
		}
		return nil
	}
}

func parseTags(flags []string) (map[string]string, error) {
	tags := make(map[string]string)
	for _, flag := range flags {
		data := strings.Split(flag, "=")
		if len(data) != 2 { //nolint:gomnd
			return nil, errors.Errorf("Invalid tags. Tags should be in a KEY=VALUE format")
		}
		tags[data[0]] = data[1]
	}
	return tags, nil
}

func runListScansCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allScansModel *scansRESTApi.ScansCollectionResponseModel
		var errorModel *scansRESTApi.ErrorModel
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
			return errors.Errorf("%s: CODE: %d, %s\n", failedGettingAll, errorModel.Code, errorModel.Message)
		} else if allScansModel != nil && allScansModel.Scans != nil {
			err = Print(cmd.OutOrStdout(), toScanViews(allScansModel.Scans))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func runGetScanByIDCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var scanResponseModel *scansRESTApi.ScanResponseModel
		var errorModel *scansRESTApi.ErrorModel
		var err error
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a scan ID", failedGetting)
		}
		scanID := args[0]
		scanResponseModel, errorModel, err = scansWrapper.GetByID(scanID)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGetting)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message)
		} else if scanResponseModel != nil {
			err = Print(cmd.OutOrStdout(), toScanView(scanResponseModel))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func runDeleteScanCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var errorModel *scansRESTApi.ErrorModel
		var err error
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a scan ID", failedDeleting)
		}
		scanID := args[0]
		errorModel, err = scansWrapper.Delete(scanID)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedDeleting)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedDeleting, errorModel.Code, errorModel.Message)
		}
		return nil
	}
}

func runGetTagsCommand(scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var tags map[string][]string
		var errorModel *scansRESTApi.ErrorModel
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
			fmt.Fprintln(cmd.OutOrStdout(), string(tagsJSON))
		}
		return nil
	}
}

type scanView struct {
	ID        string `format:"name:Scan ID"`
	ProjectID string `format:"name:Project ID"`
	Status    string
	CreatedAt time.Time `format:"name:Created at;time:06-01-02 15:04:05"`
	UpdatedAt time.Time `format:"name:Updated at;time:06-01-02 15:04:05"`
	Tags      map[string]string
}

func toScanViews(scans []scansRESTApi.ScanResponseModel) []*scanView {
	views := make([]*scanView, len(scans))
	for i := 0; i < len(scans); i++ {
		views[i] = toScanView(&scans[i])
	}
	return views
}

func toScanView(scan *scansRESTApi.ScanResponseModel) *scanView {
	return &scanView{
		ID:        scan.ID,
		Status:    string(scan.Status),
		CreatedAt: scan.CreatedAt,
		UpdatedAt: scan.UpdatedAt,
		ProjectID: scan.ProjectID,
		Tags:      scan.Tags,
	}
}
