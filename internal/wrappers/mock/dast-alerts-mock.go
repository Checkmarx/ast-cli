package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

// DastAlertsMockWrapper is a mock implementation of DastAlertsWrapper
type DastAlertsMockWrapper struct{}

// GetAlerts mocks the GetAlerts method
func (a *DastAlertsMockWrapper) GetAlerts(environmentID, scanID string, params map[string]string) (*wrappers.DastAlertsCollectionResponseModel, *wrappers.ErrorModel, error) {
	return &wrappers.DastAlertsCollectionResponseModel{
		PagesNumber: 1,
		Results: []wrappers.DastAlertResponseModel{
			{
				AlertSimilarityID: "bd9ba1ca8cca3a8b0049352a80c75f73a76fd002ab566c2f3a5da933580af035",
				State:             "To Verify",
				Severity:          "HIGH",
				Name:              "PII Disclosure",
				NumInstances:      3,
				Status:            "New",
				OWASP:             []string{"OWASP TOP 10"},
				NumNotes:          0,
				Systemic:          false,
			},
		},
		Total: 1,
	}, nil, nil
}

