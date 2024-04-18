package wrappers

import (
	"strconv"
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
)

type ResultSummary struct {
	TotalIssues     int
	HighIssues      int
	MediumIssues    int
	LowIssues       int
	InfoIssues      int
	SastIssues      int
	KicsIssues      int
	ScaIssues       int
	ScsIssues       int
	APISecurity     APISecResult
	SCSOverview     SCSOverview
	RiskStyle       string
	RiskMsg         string
	Status          string
	ScanID          string
	ScanDate        string
	ScanTime        string
	CreatedAt       string
	ProjectID       string
	BaseURI         string
	Tags            map[string]string
	ProjectName     string
	BranchName      string
	ScanInfoMessage string
	EnginesEnabled  []string
	Policies        *PolicyResponseModel
	EnginesResult   EnginesResultsSummary
}

// nolint: govet
type APISecResult struct {
	APICount         int                `json:"api_count,omitempty"`
	TotalRisksCount  int                `json:"total_risks_count,omitempty"`
	Risks            []int              `json:"risks,omitempty"`
	RiskDistribution []riskDistribution `json:"risk_distribution,omitempty"`
	StatusCode       int
}

type riskDistribution struct {
	Origin string `json:"origin,omitempty"`
	Total  int    `json:"total,omitempty"`
}

type SCSOverview struct {
	Status               ScanStatus             `json:"status"`
	TotalRisksCount      int                    `json:"totalRisks,omitempty"`
	RiskSummary          map[string]int         `json:"riskSummary,omitempty"`
	MicroEngineOverviews []*MicroEngineOverview `json:"engineOverviews,omitempty"`
}

type MicroEngineOverview struct {
	Name        string         `json:"name"`
	FullName    string         `json:"fullName"`
	Status      ScanStatus     `json:"status"`
	TotalRisks  int            `json:"totalRisks"`
	RiskSummary map[string]int `json:"riskSummary"`
}

type EngineResultSummary struct {
	High       int
	Medium     int
	Low        int
	Info       int
	StatusCode int
}

type EnginesResultsSummary map[string]*EngineResultSummary

func (engineSummary *EnginesResultsSummary) GetHighIssues() int {
	highIssues := 0
	for _, v := range *engineSummary {
		highIssues += v.High
	}
	return highIssues
}

func (engineSummary *EnginesResultsSummary) GetLowIssues() int {
	lowIssues := 0
	for _, v := range *engineSummary {
		lowIssues += v.Low
	}
	return lowIssues
}

func (engineSummary *EnginesResultsSummary) GetMediumIssues() int {
	mediumIssues := 0
	for _, v := range *engineSummary {
		mediumIssues += v.Medium
	}
	return mediumIssues
}

func (engineSummary *EnginesResultsSummary) GetInfoIssues() int {
	infoIssues := 0
	for _, v := range *engineSummary {
		infoIssues += v.Info
	}
	return infoIssues
}

func (engineSummary *EngineResultSummary) Increment(level string) {
	switch level {
	case "high":
		engineSummary.High++
	case "medium":
		engineSummary.Medium++
	case "low":
		engineSummary.Low++
	case "info":
		engineSummary.Info++
	}
}

func (r *ResultSummary) UpdateEngineResultSummary(engineType, severity string) {
	r.EnginesResult[engineType].Increment(severity)
}

func (r *ResultSummary) HasEngine(engine string) bool {
	for _, v := range r.EnginesEnabled {
		if strings.Contains(engine, v) {
			return true
		}
	}
	return false
}

func (r *ResultSummary) HasAPISecurity() bool {
	return r.HasEngine(params.APISecType)
}

func (r *ResultSummary) HasSCS() bool {
	return r.HasEngine(params.ScsType)
}

func (r *ResultSummary) getRiskFromAPISecurity(origin string) *riskDistribution {
	for _, risk := range r.APISecurity.RiskDistribution {
		if strings.EqualFold(risk.Origin, origin) {
			return &risk
		}
	}
	return nil
}

func (r *ResultSummary) HasAPISecurityDocumentation() bool {
	if len(r.APISecurity.RiskDistribution) > 1 && strings.EqualFold(r.APISecurity.RiskDistribution[1].Origin, "documentation") {
		return true
	}
	return false
}

func (r *ResultSummary) GetAPISecurityDocumentationTotal() int {
	riskAPIDocumentation := r.getRiskFromAPISecurity("documentation")
	if riskAPIDocumentation != nil {
		return riskAPIDocumentation.Total
	}
	return 0
}

func (r *ResultSummary) HasPolicies() bool {
	return r.Policies != nil && len(r.Policies.Policies) > 0
}

func (r *ResultSummary) GeneratePolicyHTML() string {
	html := `
<div class="element">
	<div class="header-policy">`
	if r.Policies.BreakBuild {
		html += "Policy Management Violation - Break Build Enabled\n"
	} else {
		html += "Policy Management Violation\n"
	}
	html += `
	</div>
	<table id="policy">
		<tr>
			<td>
				Policy
			</td>
			<td>
				Rule
			</td>
			<td>
				Break Build
			</td>
		</tr>
`
	for _, policy := range r.Policies.Policies {
		html += `<tr>
					<td>` +
			policy.Name +
			`	
					</td>
				` +
			`
				<td>
			` + strings.Join(policy.RulesViolated, ",") +
			`
				</td>
			` + `<td>` +
			strconv.FormatBool(policy.BreakBuild) +
			`
				</td>
			</tr>
			`
	}
	html += `</table>
</div>`
	return html
}

func (r *ResultSummary) GeneratePolicyMarkdown() string {
	markdown := ""
	if r.Policies.BreakBuild {
		markdown += "### Policy Management Violation - Break Build Enabled\n"
	} else {
		markdown += "### Policy Management Violation\n"
	}
	markdown += "| Policy | Rule | Break Build |\n|:----------:|:------------:|:---------:|\n"
	for _, policy := range r.Policies.Policies {
		markdown += "|" + policy.Name + "|" + strings.Join(policy.RulesViolated, ",") + "|" + strconv.FormatBool(policy.BreakBuild) + "|\n"
	}
	return markdown
}

const summaryTemplateHeader = `{{define "SummaryTemplate"}}
<!DOCTYPE html>
<html lang="en">

<head>
    <meta http-equiv="Content-type" content="text/html; charset=utf-8">
    <meta http-equiv="Content-Language" content="en-us">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <title>Checkmarx Scan Report</title>
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

		.bg-api-sec {
            background-color: #bdbdbd !important;
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
			margin-top: 10rem;
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
            margin: 0 1rem 2rem;
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
            justify-content: center;
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
            display: flex;
            justify-content: space-evenly;
            padding: 20px;
            width: 100%;
        }

		.second-row {
            -ms-flex-pack: justify;
            -webkit-box-pack: justify;
            align-items: normal;
            display: flex;
            justify-content: space-evenly;
            padding: 20px;
            width: 100%;
			height: 100px;
        }

		.second-row .element {
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
            min-height: 100px;
            padding: 16px 20px;
            width: 24.5%;
			margin: 0 3rem 2rem;
    		right: 40px;
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
        .cx-details { 
            color: black;
            align-items: center;
            text-align: center;
            margin-bottom: 10px;
        }
		#policy {
		  	border-collapse: collapse;
		  	width: 100%;
		}
		
		#policy td, #policy th {
		  	border: 1px solid #ddd;
		  	padding: 8px;
			font-size: 14px;
            width: 25%;
		}

		#policy tr{
			word-break:break-all;
		}

		#policy tr:nth-child(even){background-color: #f2f2f2;}
		
		#policy th {
		  	padding-top: 12px;
		  	padding-bottom: 12px;
		  	text-align: left;
		  	background-color: #04AA6D;
		  	color: white;
		}
		.header-policy {
			-ms-flex-pack: justify;
			-webkit-box-pack: justify;
			display: flex;
			height: 20px;
			justify-content: space-between;
			margin-bottom: 5px;
			align-items: center;
			justify-content: left;
			font-size: 15px;
			font-weight: 700;
		}
        @media only screen and (max-width: 1100px) {
            .top-row  {
            	display: block;
            }
            .element.risk-level-tile.high{
                width: 100%;
            }
            .element{
                width: 100% !important;
            }
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
                    <a href="{{.BaseURI}}" target="_blank">More details</a>
                </div>
            </div>

        </div>`

const nonAsyncSummary = `<div class="top-row">
            <div class="element risk-level-tile {{.RiskStyle}}"><span class="value">{{.RiskMsg}}</span></div>
			{{if .HasPolicies}}
				{{.GeneratePolicyHTML}}
			{{end}}
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
                    <div class="legend"><span class="engines-legend-dot">IaC Security</span>
                        <div class="severity-engines-text bg-kicks"></div>
                    </div>
                    <div class="legend"><span class="engines-legend-dot">SCA</span>
                        <div class="severity-engines-text bg-sca"></div>
                    </div>
                </div>
                <div class="chart">
                    <div class="single-stacked-bar-chart bar-chart">
                        <div class="progress">
                            <div class="progress-bar bg-sast value">{{if lt .SastIssues 0}}N/A{{else}}{{.SastIssues}}{{end}}</div>
                            <div class="progress-bar bg-kicks value">{{if lt .KicsIssues 0}}N/A{{else}}{{.KicsIssues}}{{end}}</div>
							<div class="progress-bar bg-sca value">{{if lt .ScaIssues 0}}N/A{{else}}{{.ScaIssues}}{{end}}</div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        {{if .HasAPISecurity}}
        <hr>
		<div class="second-row">
				<div class="element">
                	<div class="total">Detected APIs</div>
 					<div class="total">{{.APISecurity.APICount}}</div>
                </div>
				<div class="element">
                	<div class="total">APIs with risk</div>
 					<div class="total">{{.APISecurity.TotalRisksCount}}</div>
                </div>
		</div>
        {{end}}`

const asyncSummaryTemplate = `<div class="cx-info">
            <div class="data">
                <div class="cx-details">{{.ScanInfoMessage}}</div>
            </div>
        </div>`

const summaryTemplateFooter = `</div>
</body>
{{end}}
`

// nolint: lll
const SummaryMarkdownPendingTemplate = `
# Checkmarx One Scan Summary
***
##### {{.ScanInfoMessage}}
##### [🔗 More details]({{.BaseURI}})
***
`

// nolint: lll
const SummaryMarkdownCompletedTemplate = `
{{- /* The '-' symbol at the start of the line is used to strip leading white space */ -}}
{{- /* ResultSummary template */ -}}
{{ $emoji := "⚪" }}
{{ if eq .RiskMsg "High Risk" }}
  {{ $emoji = "🔴" }}
{{ else if eq .RiskMsg "Medium Risk" }}
  {{ $emoji = "🟡" }}
{{ else if eq .RiskMsg "Low Risk" }}
  {{ $emoji = "⚪" }}
{{ end }}
# Checkmarx One Scan Summary
***

### {{$emoji}} {{.RiskMsg}} {{$emoji}}
######  Scan : 💾 {{.ScanID}}     |   📅 {{.CreatedAt}}    |  [🔗 More details]({{.BaseURI}})
***

{{if .HasPolicies}}
{{.GeneratePolicyMarkdown}}
{{end}}

### Total Vulnerabilities: {{.TotalIssues}}

|🔴 High |🟡 Medium |⚪ Low |⚪ Info |
|:----------:|:------------:|:---------:|:----------:|
| {{.HighIssues}} | {{.MediumIssues}} | {{.LowIssues}} | {{.InfoIssues}} |
***

### Vulnerabilities per Scan Type

| SAST | IaC Security | SCA |
|:----------:|:----------:|:---------:|
| {{if lt .SastIssues 0}}N/A{{else}}{{.SastIssues}}{{end}} | {{if lt .KicsIssues 0}}N/A{{else}}{{.KicsIssues}}{{end}} | {{if lt .ScaIssues 0}}N/A{{else}}{{.ScaIssues}}{{end}} |

{{if .HasAPISecurity}}
### API Security 

| Detected APIs | APIs with risk | {{if .HasAPISecurityDocumentation}} APIs Documentation |{{end}}
|:---------:|:---------:| {{if .HasAPISecurityDocumentation}}:---------:|{{end}}
| {{.APISecurity.APICount}} | {{.APISecurity.TotalRisksCount}} | {{if .HasAPISecurityDocumentation}} {{.GetAPISecurityDocumentationTotal}} |{{end}}
{{end}}
`

func SummaryMarkdownTemplate(isScanPending bool) string {
	if isScanPending {
		return SummaryMarkdownPendingTemplate
	}

	return SummaryMarkdownCompletedTemplate
}

func SummaryTemplate(isScanPending bool) string {
	result := summaryTemplateHeader
	if !isScanPending {
		result += nonAsyncSummary
	} else {
		result += asyncSummaryTemplate
	}
	result += summaryTemplateFooter
	return result
}
