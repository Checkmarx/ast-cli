package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type RisksManagementMockWrapper struct{}

func (r *RisksManagementMockWrapper) GetTopVulnerabilitiesByProjectID(projectID string) (*wrappers.ASPMResult, *wrappers.WebError, error) {
	mockResults := []wrappers.RiskManagementResult{
		{ID: "1", Name: "Vuln1", Severity: "High"},
		{ID: "2", Name: "Vuln2", Severity: "Medium"},
	}

	mockASPMResult := &wrappers.ASPMResult{
		ProjectID: projectID,
		Results:   mockResults,
	}

	return mockASPMResult, nil, nil
}
