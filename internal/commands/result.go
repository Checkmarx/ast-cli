package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/checkmarxDev/ast-cli/internal/commands/util"

	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedListingResults = "Failed listing results"
	mediumLabel          = "medium"
	highLabel            = "high"
	lowLabel             = "low"
	sastTypeFlag         = "sast"
)

var (
	filterResultsListFlagUsage = fmt.Sprintf(
		"Filter the list of results. Use ';' as the delimeter for arrays. Available filters are: %s",
		strings.Join(
			[]string{
				commonParams.ScanIDQueryParam,
				commonParams.LimitQueryParam,
				commonParams.OffsetQueryParam,
				commonParams.SortQueryParam,
				commonParams.IncludeNodesQueryParam,
				commonParams.NodeIDsQueryParam,
				commonParams.QueryQueryParam,
				commonParams.GroupQueryParam,
				commonParams.StatusQueryParam,
				commonParams.SeverityQueryParam,
			}, ",",
		),
	)
	scanAPIPath = ""
)

func NewResultCommand(resultsWrapper wrappers.ResultsWrapper) *cobra.Command {
	scanAPIPath = resultsWrapper.GetScaAPIPath()
	resultCmd := &cobra.Command{
		Use:   "result",
		Short: "Retrieve results",
		RunE:  runGetResultCommand(resultsWrapper),
	}
	addScanIDFlag(resultCmd, "ID to report on.")
	addResultFormatFlag(resultCmd, util.FormatJSON, util.FormatSummary, util.FormatSummaryConsole, util.FormatSarif)
	resultCmd.PersistentFlags().String(TargetFlag, "", "Output file")
	resultCmd.PersistentFlags().String(TargetPathFlag, ".", "Output Path")
	resultCmd.PersistentFlags().StringSlice(FilterFlag, []string{}, filterResultsListFlagUsage)
	return resultCmd
}

func getScanInfo(scanID string) (*ResultSummary, error) {
	scansWrapper := wrappers.NewHTTPScansWrapper(scanAPIPath)
	scanInfo, errorModel, err := scansWrapper.GetByID(scanID)
	if err != nil {
		return nil, errors.Wrapf(err, "%s", failedGetting)
	}
	if errorModel != nil {
		return nil, errors.Errorf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message)
	} else if scanInfo != nil {
		return &ResultSummary{
			ScanID:       scanInfo.ID,
			Status:       string(scanInfo.Status),
			CreatedAt:    scanInfo.CreatedAt.Format("2006-01-02, 15:04:05"),
			ProjectID:    scanInfo.ProjectID,
			RiskStyle:    "",
			RiskMsg:      "",
			HighIssues:   0,
			MediumIssues: 0,
			LowIssues:    0,
			SastIssues:   0,
			KicsIssues:   0,
			ScaIssues:    0,
			Tags:         scanInfo.Tags,
		}, nil
	}
	return nil, err
}

func SummaryReport(results *wrappers.ScanResultsCollection, scanID string) (*ResultSummary, error) {
	summary, err := getScanInfo(scanID)
	if err != nil {
		return nil, err
	}
	summary.TotalIssues = int(results.TotalCount)
	for _, result := range results.Results {
		if result.Severity == "HIGH" {
			summary.HighIssues++
			summary.RiskStyle = highLabel
			summary.RiskMsg = "High Risk"
		}
		if result.Severity == "LOW" {
			summary.LowIssues++
			if summary.RiskStyle != highLabel && summary.RiskStyle != mediumLabel {
				summary.RiskStyle = lowLabel
				summary.RiskMsg = "Low Risk"
			}
		}
		if result.Severity == mediumLabel {
			summary.MediumIssues++
			if summary.RiskStyle != highLabel {
				summary.RiskStyle = mediumLabel
				summary.RiskMsg = "Medium Risk"
			}
		}
	}
	return summary, nil
}

func writeHTMLSummary(targetFile string, summary *ResultSummary) error {
	fmt.Println("Creating Summary Report: ", targetFile)
	summaryTemp, err := template.New("summaryTemplate").Parse(summaryTemplate)
	if err == nil {
		f, err := os.Create(targetFile)
		if err == nil {
			_ = summaryTemp.ExecuteTemplate(f, "SummaryTemplate", summary)
			_ = f.Close()
		}
		return err
	}
	return nil
}

func writeConsoleSummary(summary *ResultSummary) error {
	fmt.Println("")
	fmt.Printf("         Created At: %s\n", summary.CreatedAt)
	fmt.Printf("               Risk: %s\n", summary.RiskMsg)
	fmt.Printf("         Project ID: %s\n", summary.ProjectID)
	fmt.Printf("            Scan ID: %s\n", summary.ScanID)
	fmt.Printf("       Total Issues: %d\n", summary.TotalIssues)
	fmt.Printf("        High Issues: %d\n", summary.HighIssues)
	fmt.Printf("      Medium Issues: %d\n", summary.MediumIssues)
	fmt.Printf("         Low Issues: %d\n", summary.LowIssues)
	return nil
}

func runGetResultCommand(resultsWrapper wrappers.ResultsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		targetFile, _ := cmd.Flags().GetString(TargetFlag)
		targetPath, _ := cmd.Flags().GetString(TargetPathFlag)
		format, _ := cmd.Flags().GetString(TargetFormatFlag)
		scanID, _ := cmd.Flags().GetString(ScanIDFlag)
		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingResults)
		}
		return CreateScanReport(resultsWrapper, scanID, format, targetFile, targetPath, params)
	}
}

func CreateScanReport(resultsWrapper wrappers.ResultsWrapper,
	scanID string,
	reportTypes string,
	targetFile string,
	targetPath string,
	params map[string]string) error {
	if scanID == "" {
		return errors.Errorf("%s: Please provide a scan ID", failedListingResults)
	}
	results, err := ReadResults(resultsWrapper, scanID, params)
	if err != nil {
		return err
	}
	summary, err := SummaryReport(results, scanID)
	if err != nil {
		return err
	}
	reportList := strings.Split(reportTypes, ",")
	for _, reportType := range reportList {
		err = createReport(reportType, targetFile, targetPath, results, summary)
		if err != nil {
			return err
		}
	}
	return nil
}

func createReport(format string,
	targetFile string,
	targetPath string,
	results *wrappers.ScanResultsCollection,
	summary *ResultSummary) error {
	if util.IsFormat(format, util.FormatSarif) {
		sarifRpt := createTargetName(targetFile, targetPath, "sarif")
		return exportSarifResults(sarifRpt, results)
	}
	if util.IsFormat(format, util.FormatJSON) {
		jsonRpt := createTargetName(targetFile, targetPath, "json")
		return exportJSONResults(jsonRpt, results)
	}
	if util.IsFormat(format, util.FormatSummaryConsole) {
		return writeConsoleSummary(summary)
	}
	if util.IsFormat(format, util.FormatSummary) {
		summaryRpt := createTargetName(targetFile, targetPath, "html")
		return writeHTMLSummary(summaryRpt, summary)
	}
	err := fmt.Errorf("Bad report format %s", format)
	return err
}

func createTargetName(targetFile string, targetPath string, targetType string) string {
	return targetPath + "/" + targetFile + "." + targetType
}

func ReadResults(
	resultsWrapper wrappers.ResultsWrapper,
	scanID string,
	params map[string]string,
) (results *wrappers.ScanResultsCollection, err error) {
	var resultsModel *wrappers.ScanResultsCollection
	var errorModel *resultsHelpers.WebError
	params[commonParams.ScanIDQueryParam] = scanID
	resultsModel, errorModel, err = resultsWrapper.GetAllResultsByScanID(params)
	if err != nil {
		return nil, errors.Wrapf(err, "%s", failedListingResults)
	}
	if errorModel != nil {
		return nil, errors.Errorf("%s: CODE: %d, %s", failedListingResults, errorModel.Code, errorModel.Message)
	} else if resultsModel != nil {
		return resultsModel, nil
	}
	return nil, nil
}

func exportSarifResults(targetFile string, results *wrappers.ScanResultsCollection) error {
	var err error
	var resultsJSON []byte
	fmt.Println("Creating SARIF Report: ", targetFile)
	var sarifResults *wrappers.SarifResultsCollection = convertCxResultsToSarif(results)
	resultsJSON, err = json.Marshal(sarifResults)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingAll)
	}
	f, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to create target file  ", failedGettingAll)
	}
	_, _ = fmt.Fprintln(f, string(resultsJSON))
	f.Close()
	return nil
}

func exportJSONResults(targetFile string, results *wrappers.ScanResultsCollection) error {
	var err error
	var resultsJSON []byte
	fmt.Println("Creating JSON Report: ", targetFile)
	resultsJSON, err = json.Marshal(results)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingAll)
	}
	f, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to create target file  ", failedGettingAll)
	}
	_, _ = fmt.Fprintln(f, string(resultsJSON))
	f.Close()
	return nil
}

func convertCxResultsToSarif(results *wrappers.ScanResultsCollection) *wrappers.SarifResultsCollection {
	var sarif *wrappers.SarifResultsCollection = new(wrappers.SarifResultsCollection)
	sarif.Schema = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json"
	sarif.Version = "2.1.0"
	sarif.Runs = []wrappers.SarifRun{}
	sarif.Runs = append(sarif.Runs, createSarifRun(results))
	return sarif
}

func createSarifRun(results *wrappers.ScanResultsCollection) wrappers.SarifRun {
	var sarifRun wrappers.SarifRun
	sarifRun.Tool.Driver.Name = "Checkmarx AST"
	sarifRun.Tool.Driver.Version = "1.0"
	sarifRun.Tool.Driver.Rules = findSarifRules(results)
	sarifRun.Results = findSarifResults(results)
	return sarifRun
}

func findSarifRules(results *wrappers.ScanResultsCollection) []wrappers.SarifDriverRule {
	var sarifRules = []wrappers.SarifDriverRule{}
	if results == nil {
		return sarifRules
	}
	for _, result := range results.Results {
		if result.Type == sastTypeFlag {
			continue
		}
		var sarifRule wrappers.SarifDriverRule
		sarifRule.ID = result.ScanResultData.QueryID
		sarifRule.Name = result.ScanResultData.QueryName
		sarifRules = append(sarifRules, sarifRule)
	}
	return sarifRules
}

func findSarifResults(results *wrappers.ScanResultsCollection) []wrappers.SarifScanResult {
	var sarifResults = []wrappers.SarifScanResult{}
	if results == nil {
		return sarifResults
	}
	for _, result := range results.Results {
		if result.Type != sastTypeFlag {
			continue
		}

		var scanResult wrappers.SarifScanResult
		scanResult.RuleID = result.ScanResultData.QueryID
		// Mock Fix
		// scanResult.Message.Text = result.Comments
		scanResult.PartialFingerprints.PrimaryLocationLineHash = result.SimilarityID
		scanResult.Locations = []wrappers.SarifLocation{}
		var scanLocation wrappers.SarifLocation
		// TODO: when there is real data we need to find the source of the result from Result.Data.Nodes
		// this is placeholder code
		scanLocation.PhysicalLocation.ArtifactLocation.URI = ""
		scanLocation.PhysicalLocation.Region.StartLine = result.ScanResultData.Nodes[0].Line
		// TODO: fix this column issue and places that reference it when
		// the data structures are fixed.
		// column := result.ScanResultData.Nodes[0].Column
		var column uint = 0
		length := result.ScanResultData.Nodes[0].Length
		scanLocation.PhysicalLocation.Region.StartColumn = column
		scanLocation.PhysicalLocation.Region.EndColumn = column + length
		scanResult.Locations = append(scanResult.Locations, scanLocation)
		sarifResults = append(sarifResults, scanResult)
	}
	return sarifResults
}

func outputSingleResult(w io.Writer, model *wrappers.ScanResult) {
	_, _ = fmt.Fprintln(w, "Result Unique ID:", model.ScanResultData.PackageID)
	_, _ = fmt.Fprintln(w, "Query ID:", model.ScanResultData.QueryID)
	_, _ = fmt.Fprintln(w, "Query Name:", model.ScanResultData.QueryName)
	_, _ = fmt.Fprintln(w, "Severity:", model.Severity)
	_, _ = fmt.Fprintln(w, "Similarity ID:", model.SimilarityID)
	_, _ = fmt.Fprintln(w, "First Scan ID:", model.FirstScanID)
	_, _ = fmt.Fprintln(w, "Found At:", model.FoundAt)
	_, _ = fmt.Fprintln(w, "First Found At:", model.FirstFoundAt)
	_, _ = fmt.Fprintln(w, "Status:", model.Status)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "************ Nodes ************")
}

type ResultSummary struct {
	TotalIssues  int
	HighIssues   int
	MediumIssues int
	LowIssues    int
	SastIssues   int
	KicsIssues   int
	ScaIssues    int
	RiskStyle    string
	RiskMsg      string
	Status       string
	ScanID       string
	ScanDate     string
	ScanTime     string
	CreatedAt    string
	ProjectID    string
	Tags         map[string]string
}

const summaryTemplate = `
{{define "SummaryTemplate"}}
<!DOCTYPE html>
<html lang="en">

<head>
    <meta http-equiv="Content-type" content="text/html; charset=utf-8">
    <meta http-equiv="Content-Language" content="en-us">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <title>Checkmarx test report</title>
    <style type="text/css">
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        .bg-green {
            background-color: #f9ae4d;
        }

        .bg-grey {
            background-color: #bdbdbd;
        }

        .bg-kicks {
            background-color: #008e96 !important;
        }

        .bg-red {
            background-color: #f1605d;
        }

        .bg-sast {
            background-color: #1165b4 !important;
        }

        .bg-sca {
            background-color: #0fcdc2 !important;
        }

        .header-row .cx-info .data .calendar-svg {
            margin-right: 8px;
        }

        .header-row .cx-info .data .scan-svg svg {
            -webkit-transform: scale(0.43);
            margin-top: -9px;
            transform: scale(0.43);
        }

        .header-row .cx-info .data .scan-svg {
            margin-left: -10px;
        }

        .header-row .cx-info .data svg path {
            fill: #565360;
        }

        .header-row .cx-info .data {
            color: #565360;
            display: flex;
            margin-right: 20px;
        }

        .header-row .cx-info {
            display: flex;
            font-size: 13px;
        }

        .header-row {
            -ms-flex-pack: justify;
            -webkit-box-pack: justify;
            display: flex;
            height: 30px;
            justify-content: space-between;
            margin-bottom: 5px;
            align-items: center;
            justify-content: center;
        }

        .cx-cx-main {
            align-items: center;
            display: flex;
            flex-flow: row wrap;
            justify-content: space-around;
            left: 0;
            position: relative;
            top: 0;
            width: 100%;
						margin-top: 10rem
        }

        .progress {
            background-color: #e9ecef;
            display: flex;
            height: 1em;
            overflow: hidden;
        }

        .progress-bar {
            -ms-flex-direction: column;
            -ms-flex-pack: center;
            -webkit-box-direction: normal;
            -webkit-box-orient: vertical;
            -webkit-box-pack: center;
            background-color: grey;
            color: #FFF;
            display: flex;
            flex-direction: column;
            font-size: 11px;
            justify-content: center;
            text-align: center;
            white-space: nowrap;
        }


        .top-row .element {
            margin: 0 3rem 6rem;
        }

        .top-row .risk-level-tile .value {
            display: inline-block;
            font-size: 32px;
            font-weight: 700;
            margin-top: 20px;
            text-align: center;
            width: 100%;
        }

        .top-row .risk-level-tile {
            -webkit-box-shadow: 0 2px 4px rgba(0, 0, 0, 0.15);
            background: #fff;
            border: 1px solid #dad8dc;
            border-radius: 4px;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.15);
            color: #565360;
            min-height: 120px;
            width: 24.5%;
        }

        .top-row .risk-level-tile.high {
            background: #f1605d;
            color: #fcfdff;
        }

				.top-row .risk-level-tile.medium {
					background-color: #f9ae4d;
					color: #fcfdff;
        }

				.top-row .risk-level-tile.low {
					background-color: #bdbdbd;
					color: #fcfdff;
				}

        .chart .total {
            font-size: 24px;
            font-weight: 700;
        }

        .chart .bar-chart {
            margin-left: 10px;
            padding-top: 7px;
            width: 100%;
        }

        .legend {
            color: #95939b;
            float: left;
            padding-right: 10px;
            text-transform: capitalize;
        }


        .chart .total {
            font-size: 24px;
            font-weight: 700;
        }

        .chart .bar-chart {
            margin-left: 10px;
            padding-top: 7px;
            width: 100%;
        }

        .top-row .vps-tile .legend {
            color: #95939b;
            float: left;
            padding-right: 10px;
            text-transform: capitalize;
        }

        .chart .engines-bar-chart {
            margin-bottom: 6px;
            margin-top: 7px;
            width: 100%;
        }

        .legend {
            color: #95939b;
            text-transform: capitalize;
            float: left;
            padding-right: 10px;
        }

        .top-row {
            -ms-flex-pack: justify;
            -webkit-box-pack: justify;
            align-items: center;
            display: flex;
            justify-content: space-evenly;
            padding: 20px;
            width: 100%;
        }

        .bar-chart .progress .progress-bar.bg-danger {
            background-color: #f1605d !important;
        }

        .bar-chart .progress .progress-bar.bg-success {
            background-color: #bdbdbd !important;
        }

        .bar-chart .progress .progress-bar.bg-warning {
            background-color: #f9ae4d !important;
        }

        .width-100 {
            width: 100%;
        }

        .bar-chart .progress .progress-bar {
            color: #FFF;
            font-size: 11px;
            font-weight: 500;
            min-width: fit-content;
            padding: 0 3px;
        }

        .bar-chart .progress .progress-bar:not(:last-child),
        .progress-bar:not(:last-child),
        .bar-chart .progress .progress-bar:not(:last-child) {
            border-right: 1px solid #FFF;
        }

        .bar-chart .progress {
            background: url(data:image/png;base64,iVBORw0KGgoAAAANSUhE
							UgAAAAQAAAAECAYAAACp8Z5+AAAAIklEQVQYV2NkQAIfPnz6zwjjgzgCAny
							MYAEYB8RmROaABAAU7g/W6mdTYAAAAABJRU5ErkJggg==) repeat;
            border: 1px solid #f0f0f0;
            border-radius: 3px;
            height: 1.5rem;
            overflow: hidden;
        }

        .engines-legend-dot,
        .severity-legend-dot {
            font-size: 14px;
            padding-left: 5px;
        }

        .severity-engines-text,
        .severity-legend-text {
            float: left;
            height: 10px;
            margin-top: 5px;
            width: 10px;
        }

        .chart {
            display: flex;
        }

        .element .total {
            font-weight: 700;
        }

        .top-row .element {
            -ms-flex-direction: column;
            -ms-flex-pack: justify;
            -webkit-box-direction: normal;
            -webkit-box-orient: vertical;
            -webkit-box-pack: justify;
            -webkit-box-shadow: 0 2px 4px rgba(0, 0, 0, 0.15);
            background: #fff;
            border-radius: 4px;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.15);
            color: #565360;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            min-height: 120px;
            padding: 16px 20px;
            width: 24.5%;
        }
        .cx-demo { 
            color: red;
            align-items: center;
            text-align: center;
            margin-bottom: 10px;
        }
    </style>
    <script>
        window.addEventListener('load', function () {
            var totalVal = document.getElementById("total").textContent;
            var elements = document.getElementsByClassName("value");
            for (var i = 0; i < elements.length; i++) {
                var element = elements[i];
                var perc = ((element.textContent / totalVal) * 100);
                element.style.width = perc + "%";
            }
        }, false);
    </script>
</head>

<body>
		<br/>
		<br/>
    <div class="cx-main">
        <div class="header-row">
            <div class="cx-info">
                <div class="data">
                    <div class="scan-svg"><svg width="40" height="40" fill="none">
                            <path fill-rule="evenodd" clip-rule="evenodd"
                                d="M9.393 32.273c-.65.651-1.713.656-2.296-.057A16.666 
																16.666 0 1136.583 20h1.75v3.333H22.887a3.333 3.333 0 110-3.333h3.911a7 7 0 10-12.687 
																5.45c.447.698.464 1.641-.122 2.227-.586.586-1.546.591-2.038-.075A10 
																10 0 1129.86 20h3.368a13.331 13.331 0 00-18.33-10.652A13.334 13.334 0 009.47 
																29.846c.564.727.574 1.776-.077 2.427z"
                                fill="url(#scans_svg__paint0_angular)"></path>
                            <path fill-rule="evenodd" clip-rule="evenodd"
                                d="M9.393 32.273c-.65.651-1.713.656-2.296-.057A16.666 16.666 0 1136.583 
																20h1.75v3.333H22.887a3.333 3.333 0 110-3.333h3.911a7 7 0 10-12.687 
																5.45c.447.698.464 1.641-.122 2.227-.586.586-1.546.591-2.038-.075A10 
																10 0 1129.86 20h3.368a13.331 13.331 0 00-18.33-10.652A13.334 13.334 0 
																009.47 29.846c.564.727.574 1.776-.077 2.427z"
                                fill="url(#scans_svg__paint1_angular)"></path>
                            <defs>
                                <radialGradient id="scans_svg__paint0_angular" cx="0" cy="0" r="1"
                                    gradientUnits="userSpaceOnUse"
                                    gradientTransform="matrix(1 16.50003 -16.50003 1 20 21.5)">
                                    <stop offset="0.807" stop-color="#2991F3"></stop>
                                    <stop offset="1" stop-color="#2991F3" stop-opacity="0"></stop>
                                </radialGradient>
                                <radialGradient id="scans_svg__paint1_angular" cx="0" cy="0" r="1"
                                    gradientUnits="userSpaceOnUse"
                                    gradientTransform="matrix(1 16.50003 -16.50003 1 20 21.5)">
                                    <stop offset="0.807" stop-color="#2991F3"></stop>
                                    <stop offset="1" stop-color="#2991F3" stop-opacity="0"></stop>
                                </radialGradient>
                            </defs>
                        </svg></div>
                    <div>Scan: {{.ScanID}}</div>
                </div>
                <div class="data">
                    <div class="calendar-svg"><svg width="12" height="12" fill="none">
                            <path fill-rule="evenodd" clip-rule="evenodd"
                                d="M3.333 0h1.334v1.333h2.666V0h1.334v1.333h2c.368 0 .666.299.666.667v8.667a.667.667 
																0 01-.666.666H1.333a.667.667 0 01-.666-.666V2c0-.368.298-.667.666-.667h2V0zm4 
																2.667V4h1.334V2.667H10V10H2V2.667h1.333V4h1.334V2.667h2.666z"
                                fill="#95939B"></path>
                        </svg></div>
                    <div>{{.CreatedAt}}</div>
                </div>
                
                <div class="data">
                    <a href="https://ast-master.dev.cxast.net/#/projects/{{.ProjectID}}/overview" target="_blank">More details</a>
                </div>
            </div>

        </div>
        <div class="top-row">
            <div class="element risk-level-tile {{.RiskStyle}}"><span class="value">{{.RiskMsg}}</span></div>
            <div class="element">
                <div class="total">Total Vulnerabilities</div>
                <div>
                    <div class="legend"><span class="severity-legend-dot">high</span>
                        <div class="severity-legend-text bg-red"></div>
                    </div>
                    <div class="legend"><span class="severity-legend-dot">medium</span>
                        <div class="severity-legend-text bg-green"></div>
                    </div>
                    <div class="legend"><span class="severity-legend-dot">low</span>
                        <div class="severity-legend-text bg-grey"></div>
                    </div>
                </div>
                <div class="chart">
                    <div id="total" class="total">{{.TotalIssues}}</div>
                    <div class="single-stacked-bar-chart bar-chart">
                        <div class="progress">
                            <div class="progress-bar bg-danger value">{{.HighIssues}}</div>
                            <div class="progress-bar bg-warning value">{{.MediumIssues}}</div>
                            <div class="progress-bar bg-success value">{{.LowIssues}}</div>
                        </div>
                    </div>
                </div>
            </div>
            <div class="element">
                <div class="total">Vulnerabilities per Scan Type</div>
                <div class="legend">
                    <div class="legend"><span class="engines-legend-dot">SAST</span>
                        <div class="severity-engines-text bg-sast"></div>
                    </div>
                    <div class="legend"><span class="engines-legend-dot">KICS</span>
                        <div class="severity-engines-text bg-kicks"></div>
                    </div>
                    <div class="legend"><span class="engines-legend-dot">SCA</span>
                        <div class="severity-engines-text bg-sca"></div>
                    </div>
                </div>
                <div class="chart">
                    <div class="single-stacked-bar-chart bar-chart">
                        <div class="progress">
                            <div class="progress-bar bg-sast value">{{.SastIssues}}</div>
                            <div class="progress-bar bg-kicks value">{{.KicsIssues}}</div>
														<div class="progress-bar bg-sca value">{{.ScaIssues}}</div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

    </div>
</body>
{{end}}
`
