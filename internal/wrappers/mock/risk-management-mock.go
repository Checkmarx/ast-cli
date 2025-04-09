package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type RiskManagementMockWrapper struct{}

func (r *RiskManagementMockWrapper) GetTopVulnerabilitiesByProjectID(projectID string, scanID string) (*wrappers.ASPMResult, *wrappers.WebError, error) {
	mockResults := []wrappers.RiskManagementResult{
		{ID: "1", Name: "Vuln1", Severity: "High", Traits: map[string]string{wrappers.ExpPathKey: wrappers.ExpPathValue}},
		{ID: "2", Name: "Vuln2", Severity: "Medium", Traits: map[string]string{wrappers.SuspMalwareKey: wrappers.SuspMalwareValue}},
	}

	mockASPMResult := &wrappers.ASPMResult{
		ProjectID: projectID,
		ScanID:    scanID,
		Results:   mockResults,
	}

	return mockASPMResult, nil, nil
}
