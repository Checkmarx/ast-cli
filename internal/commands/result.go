package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	resultsReader "github.com/checkmarxDev/sast-results/pkg/reader"
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	resultsRaw "github.com/checkmarxDev/sast-results/pkg/web/path/raw"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedListingResults = "Failed listing results"
	targetFlag           = "target"
	templateFlag         = "template"
	exportTemplateFlag   = "template-export"
	templateFileName     = "summary.tpl"
)

type ScanResults struct {
	Version string       `json:"version"`
	Results []ScanResult `json:"results"`
}

type ScanResult struct {
	ID             string         `json:"id"`
	SimilarityID   string         `json:"similarityId"`
	Severity       string         `json:"severity"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	State          string         `json:"state"`
	ScanResultData ScanResultData `json:"data"`
}

type ScanResultData struct {
	Comments  string               `json:"comments"`
	QueryName string               `json:"queryName"`
	Nodes     []ScanResultDataNode `json:"nodes"`
}

type ScanResultDataNode struct {
	Column     string `json:"column"`
	FileName   string `json:"fileName"`
	FullName   string `json:"fullName"`
	Name       string `json:"name"`
	Line       string `json:"line"`
	MethodLine string `json:"methodLine"`
}

type SimpleScanResult struct {
	ID             string `json:"id"`
	SimilarityID   string `json:"similarityId"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	State          string `json:"state"`
	ScanResultData string `json:"data"`
	Severity       string `json:"severity"`
	Column         string `json:"column"`
	FileName       string `json:"fileName"`
	FullName       string `json:"fullName"`
	Name           string `json:"name"`
	Line           string `json:"line"`
	MethodLine     string `json:"methodLine"`
	Comments       string `json:"comments"`
	QueryName      string `json:"queryName"`
}

var (
	filterResultsListFlagUsage = fmt.Sprintf("Filter the list of results. Use ';' as the delimeter for arrays. Available filters are: %s",
		strings.Join([]string{
			commonParams.ScanIDQueryParam,
			commonParams.LimitQueryParam,
			commonParams.OffsetQueryParam,
			commonParams.SortQueryParam,
			commonParams.IncludeNodesQueryParam,
			commonParams.NodeIDsQueryParam,
			commonParams.QueryQueryParam,
			commonParams.GroupQueryParam,
			commonParams.StatusQueryParam,
			commonParams.SeverityQueryParam}, ","))
)

func NewResultCommand(resultsWrapper wrappers.ResultsWrapper) *cobra.Command {
	resultCmd := &cobra.Command{
		Use:   "result",
		Short: "Retrieve results",
	}

	listResultsCmd := &cobra.Command{
		Use:   "list <scan-id>",
		Short: "List results for a given scan",
		RunE:  runGetResultByScanIDCommand(resultsWrapper),
	}
	listResultsCmd.PersistentFlags().StringSlice(filterFlag, []string{}, filterResultsListFlagUsage)
	addFormatFlag(listResultsCmd, formatList, formatJSON)
	addScanIDFlag(listResultsCmd, "ID to report on.")

	listSimpleResultsCmd := &cobra.Command{
		Use:   "list-simple <scan-id>",
		Short: "List 'simple' results for a given scan",
		RunE:  runGetSimpleResultByScanIDCommand(resultsWrapper),
	}
	listSimpleResultsCmd.PersistentFlags().String(targetFlag, "./simple-results.json", "Output file")

	summaryCmd := &cobra.Command{
		Use:   "summary",
		Short: "Creates summary report for scan",
		RunE:  runGetSummaryByScanIDCommand(resultsWrapper),
	}
	addFormatFlag(summaryCmd, formatHTML, formatText)
	addScanIDFlag(summaryCmd, "ID to report on.")
	summaryCmd.PersistentFlags().String(targetFlag, "console", "Output file")

	resultCmd.AddCommand(listResultsCmd, listSimpleResultsCmd, summaryCmd)
	return resultCmd
}

func runGetSummaryByScanIDCommand(resultsWrapper wrappers.ResultsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		targetFile, _ := cmd.Flags().GetString(targetFlag)
		//results, _ := os.ReadFile("mock-results.json")
		//var scanResults = ScanResults{}
		//_ = json.Unmarshal(results, &scanResults)
		//createSimpleResults(scanResults, targetFile)
		//t, err := template.New("foo").Parse(`{{define "T"}}Hello, {{.}}!{{end}}`)
		//template, err := readSummaryTemplate(templateFile, exportTemplateFlag)
		writeSummary(targetFile)
		return nil
	}
}

func writeSummary(targetFile string) {
	template, err := template.New("summaryTemplate").Parse(summaryTemplate)
	if err == nil {
		if targetFile == "console" {
			buffer := new(bytes.Buffer)
			err = template.ExecuteTemplate(buffer, "T", "<script>alert('This is AST!')</script>")
			fmt.Println(buffer)
		} else {
			f, err := os.Create(targetFile)
			if err == nil {
				err = template.ExecuteTemplate(f, "T", "<script>alert('This is AST!')</script>")
				f.Close()
			}
		}
	}
}

func runGetSimpleResultByScanIDCommand(resultsWrapper wrappers.ResultsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// TODO: Get the JSON report from AST, not mock file .....
		fmt.Println("Creating simple report (CURRENTLY MOCKED!)")
		targetFile, _ := cmd.Flags().GetString(targetFlag)
		fmt.Println("Target File: ", targetFile)
		results, _ := os.ReadFile("mock-results.json")
		var scanResults = ScanResults{}
		_ = json.Unmarshal(results, &scanResults)
		createSimpleResults(scanResults, targetFile)
		return nil
	}
}

func createSimpleResults(results ScanResults, targetFile string) {
	var simpleResults []SimpleScanResult
	for idx := range results.Results {
		result := results.Results[idx]
		simpleResult := SimpleScanResult{}
		simpleResult.ID = result.ID
		simpleResult.SimilarityID = result.SimilarityID
		simpleResult.Type = result.Type
		simpleResult.Severity = result.Severity
		simpleResult.Status = result.Status
		simpleResult.State = result.State
		simpleResult.Comments = result.ScanResultData.Comments
		simpleResult.QueryName = result.ScanResultData.QueryName
		if len(result.ScanResultData.Nodes) > 0 {
			simpleResult.Column = result.ScanResultData.Nodes[0].Column
			simpleResult.FileName = result.ScanResultData.Nodes[0].FileName
			simpleResult.FullName = result.ScanResultData.Nodes[0].FullName
			simpleResult.Name = result.ScanResultData.Nodes[0].Name
			simpleResult.Line = result.ScanResultData.Nodes[0].Line
			simpleResult.MethodLine = result.ScanResultData.Nodes[0].MethodLine
		}
		simpleResults = append(simpleResults, simpleResult)
	}
	// Write results to JSON file
	simpleResultsJSON, _ := json.Marshal(simpleResults)
	err := os.WriteFile(targetFile, simpleResultsJSON, 0666)
	if err != nil {
		fmt.Println("Error writing to output file.")
	}
}

func runGetResultByScanIDCommand(resultsWrapper wrappers.ResultsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var sastResultResponseModel *resultsRaw.ResultsCollection
		var kicsResultResponseModel *resultsRaw.ResultsCollection
		var resultResponseModel *resultsRaw.ResultsCollection
		var errorModel *resultsHelpers.WebError
		var kicsErrorModel *resultsHelpers.WebError
		var err error
		var kicsErr error
		scanID, _ := cmd.Flags().GetString(scanIDFlag)
		if scanID == "" {
			return errors.Errorf("%s: Please provide a scan ID", failedListingResults)
		}
		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingResults)
		}
		params[commonParams.ScanIDQueryParam] = scanID
		sastResultResponseModel, errorModel, err = resultsWrapper.GetSastByScanID(params)
		kicsResultResponseModel, kicsErrorModel, kicsErr = resultsWrapper.GetKicsByScanID(params)
		if err != nil && kicsErr != nil {
			return errors.Wrapf(err, "%s", failedListingResults)
		}
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedListingResults, errorModel.Code, errorModel.Message)
		} else if kicsErrorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedListingResults, kicsErrorModel.Code, kicsErrorModel.Message)
		} else if sastResultResponseModel != nil && kicsResultResponseModel != nil {
			resultResponseModel = mergeResults(sastResultResponseModel, kicsResultResponseModel)
			return exportResults(cmd, resultResponseModel)
		}
		fmt.Println("Nothing to do")
		return nil
	}
}

// Merges all results into the sast collection, currently only supports sast and kics
func mergeResults(sast *resultsRaw.ResultsCollection, kics *resultsRaw.ResultsCollection) *resultsRaw.ResultsCollection {
	sast.Results = append(sast.Results, kics.Results...)
	sast.TotalCount += kics.TotalCount
	return sast
}

func exportResults(cmd *cobra.Command, results *resultsRaw.ResultsCollection) error {
	var err error
	formatFlag, _ := cmd.Flags().GetString(formatFlag)
	if IsFormat(formatFlag, formatJSON) {
		var resultsJSON []byte
		resultsJSON, err = json.Marshal(results)
		if err != nil {
			return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingAll)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(resultsJSON))
		return nil
	}
	return outputResultsPretty(results.Results)
}

func outputResultsPretty(results []*resultsReader.Result) error {
	fmt.Println("************ Results ************")
	for i := 0; i < len(results); i++ {
		outputSingleResult(&resultsReader.Result{
			ResultQuery: resultsReader.ResultQuery{
				QueryID:   results[i].QueryID,
				QueryName: results[i].QueryName,
				Severity:  results[i].Severity,
				CweID:     results[i].CweID,
			},
			SimilarityID: results[i].SimilarityID,
			UniqueID:     results[i].UniqueID,
			FirstScanID:  results[i].FirstScanID,
			FirstFoundAt: results[i].FirstFoundAt,
			FoundAt:      results[i].FoundAt,
			Status:       results[i].Status,
			PathSystemID: results[i].PathSystemID,
			Nodes:        results[i].Nodes,
		})
		fmt.Println()
	}
	return nil
}

func outputSingleResult(model *resultsReader.Result) {
	fmt.Println("Result Unique ID:", model.UniqueID)
	fmt.Println("Query ID:", model.QueryID)
	fmt.Println("Query Name:", model.QueryName)
	fmt.Println("Severity:", model.Severity)
	fmt.Println("CWE ID:", model.CweID)
	fmt.Println("Similarity ID:", model.SimilarityID)
	fmt.Println("First Scan ID:", model.FirstScanID)
	fmt.Println("Found At:", model.FoundAt)
	fmt.Println("First Found At:", model.FirstFoundAt)
	fmt.Println("Status:", model.Status)
	fmt.Println("Path System ID:", model.PathSystemID)
	fmt.Println()
	fmt.Println("************ Nodes ************")
	for i := 0; i < len(model.Nodes); i++ {
		outputSingleResultNodePretty(model.Nodes[i])
		fmt.Println()
	}
}

const summaryTemplate = `{{define "T"}}Hello, {{.}}!{{end}}`
