package mock

import (
	"fmt"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type ScaRealTimeHTTPMockWrapper struct {
	path string
}

func (s ScaRealTimeHTTPMockWrapper) GetScaVulnerabilitiesPackages(scaRequest []wrappers.ScaDependencyBodyRequest) (
	[]wrappers.ScaVulnerabilitiesResponseModel, *wrappers.WebError, error,
) {
	fmt.Println(s.path)
	fmt.Println(scaRequest)
	return []wrappers.ScaVulnerabilitiesResponseModel{
		{
			PackageName:     "org.apiguardian:apiguardian-api",
			PackageManager:  "Maven",
			Version:         "1.1.2",
			Vulnerabilities: []*wrappers.Vulnerability{},
		},
		{
			PackageName:    "junit:junit",
			PackageManager: "Maven",
			Version:        "4.10",
			Vulnerabilities: []*wrappers.Vulnerability{
				{
					Cve:                  "CVE-2020-15250",
					VulnerabilityVersion: 4,
					Description:          "In JUnit4 from version 4.7 and before 4.13.1, the test rule TemporaryFolder...",
					Type:                 "Regular",
					Cvss2:                nil,
					Cwe:                  "CWE-732",
					Severity:             "Medium",
					AffectedOss:          nil,
				},
			},
		},
	}, nil, nil
}
