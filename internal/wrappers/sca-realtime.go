package wrappers

import "time"

type ScaDependencyBodyRequest struct {
	PackageName    string
	Version        string
	PackageManager string
}

type ScaVulnerabilitiesResponseModel struct {
	FileName        string
	PackageName     string           `json:"packageName"`
	PackageManager  string           `json:"packageManager"`
	Version         string           `json:"version"`
	Vulnerabilities []*Vulnerability `json:"vulnerabilities"`
}

type Vulnerability struct {
	Cve                  string      `json:"cve"`
	VulnerabilityVersion int         `json:"vulnerabilityVersion"`
	Description          string      `json:"description"`
	Type                 string      `json:"type"`
	Cvss2                interface{} `json:"cvss2"`
	Cvss3                struct {
		PrivilegesRequired         string      `json:"privilegesRequired"`
		UserInteraction            string      `json:"userInteraction"`
		Scope                      string      `json:"scope"`
		Integrity                  string      `json:"integrity"`
		BaseScore                  string      `json:"baseScore"`
		AttackVector               string      `json:"attackVector"`
		AttackComplexity           string      `json:"attackComplexity"`
		Confidentiality            string      `json:"confidentiality"`
		Availability               string      `json:"availability"`
		ExploitCodeMaturity        interface{} `json:"exploitCodeMaturity"`
		RemediationLevel           interface{} `json:"remediationLevel"`
		ReportConfidence           interface{} `json:"reportConfidence"`
		ConfidentialityRequirement interface{} `json:"confidentialityRequirement"`
		IntegrityRequirement       interface{} `json:"integrityRequirement"`
		AvailabilityRequirement    interface{} `json:"availabilityRequirement"`
		Severity                   string      `json:"severity"`
	} `json:"cvss3"`
	Cwe         string                   `json:"cwe"`
	Published   time.Time                `json:"published"`
	UpdateTime  time.Time                `json:"updateTime"`
	Severity    string                   `json:"severity"`
	AffectedOss interface{}              `json:"affectedOss"`
	References  []*ScanResultPackageData `json:"references"`
	Created     time.Time                `json:"created"`
	Credit      interface{}              `json:"credit"`
	CreditGUID  interface{}              `json:"creditGuid"`
	Kev         interface{}              `json:"kev"`
	ExploitDB   interface{}              `json:"exploitDb"`
}

type ScaRealTimeWrapper interface {
	GetScaVulnerabilitiesPackages(scaRequest []ScaDependencyBodyRequest) ([]ScaVulnerabilitiesResponseModel, *WebError, error)
}
