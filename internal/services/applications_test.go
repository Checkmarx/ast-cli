package services

import (
	"reflect"
	"strings"
	"testing"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

const (
	mockApplicationName = "MOCK"
	testProjectName     = "test-project"
)

func Test_createApplicationIds(t *testing.T) {
	type args struct {
		applicationID          []string
		existingApplicationIds []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "When adding new application IDs, add them to all applications",
			args: args{
				applicationID:          []string{"3", "4"},
				existingApplicationIds: []string{"1", "2"}},
			want: []string{"1", "2", "3", "4"}},
		{name: "When adding existing application IDs, do not re-add them",
			args: args{
				applicationID:          []string{"1"},
				existingApplicationIds: []string{"1", "2", "3"}},
			want: []string{"1", "2", "3"}},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := createApplicationIds(ttt.args.applicationID, ttt.args.existingApplicationIds); !reflect.DeepEqual(got, ttt.want) {
				t.Errorf("createApplicationIds() = %v, want %v", got, ttt.want)
			}
		})
	}
}
func Test_ProjectAssociation_ToApplicationDirectly(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}

	tests := []struct {
		description     string
		applicationName string
		projectName     string
		error           string
	}{
		{"Project association to  Application  should  fail with 403 forbidden permission error", mock.FakeForbidden403, "random-project", errorConstants.NoPermissionToUpdateApplication},
		{"Project association to  Application should  fail with 401 unauthorized  error", mock.FakeUnauthorized401, "random-project", errorConstants.StatusUnauthorized},
		{"Project association to  Application should  fail with 400  BadRequest  error", mock.FakeBadRequest400, "random-project", errorConstants.FailedToUpdateApplication},
	}

	for _, test := range tests {
		tt := test
		t.Run(tt.description, func(t *testing.T) {
			err := associateProjectToApplication(tt.applicationName, tt.projectName, []string{}, applicationWrapper)
			assert.Assert(t, strings.Contains(err.Error(), tt.error), err.Error())
		})
	}
}

func Test_ProjectAssociation_ToApplicationWithoutDirectAssociation(t *testing.T) {
	applicationModel := wrappers.ApplicationConfiguration{}
	applicationWrapper := &mock.ApplicationsMockWrapper{}

	tests := []struct {
		description   string
		applicationID string
		projectName   string
		error         string
	}{
		{"Application update should  fail with 403 forbidden permission error", mock.FakeForbidden403, "random-project", errorConstants.NoPermissionToUpdateApplication},
		{"Application update  should  fail with 401 unauthorized  error", mock.FakeUnauthorized401, "random-project", errorConstants.StatusUnauthorized},
		{"Application update should  fail with 400  BadRequest  error", mock.FakeBadRequest400, "random-project", errorConstants.FailedToUpdateApplication},
	}

	for _, test := range tests {
		tt := test
		t.Run(tt.description, func(t *testing.T) {
			err := updateApplication(&applicationModel, applicationWrapper, tt.applicationID)
			assert.Assert(t, strings.Contains(err.Error(), tt.error), err.Error())
		})
	}
}

func Test_FindApplicationAndUpdate_ProjectAlreadyAssociated_FlagEnabled(t *testing.T) {
	applicationName := mockApplicationName
	projectID := "ProjectID1" // This ID is already in the mock application's ProjectIds
	projectName := testProjectName

	// Setup mocks
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	tenantWrapper := &mock.TenantConfigurationMockWrapper{}

	// Set flag to ENABLED
	mock.Flag = wrappers.FeatureFlagResponseModel{
		Name:   "directAssociationEnabled",
		Status: true,
	}
	mock.Flags = wrappers.FeatureFlagsResponseModel{} // Empty to use single Flag

	err := findApplicationAndUpdate(applicationName, applicationWrapper, projectName, projectID, featureFlagsWrapper, tenantWrapper)
	assert.NilError(t, err)
}

func Test_FindApplicationAndUpdate_ProjectAlreadyAssociated_FlagDisabled(t *testing.T) {
	applicationName := mockApplicationName
	projectID := "ProjectID2" // This ID is already in the mock application's ProjectIds
	projectName := testProjectName

	// Setup mocks
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	tenantWrapper := &mock.TenantConfigurationMockWrapper{}

	mock.Flag = wrappers.FeatureFlagResponseModel{
		Name:   "directAssociationEnabled",
		Status: false,
	}
	mock.Flags = wrappers.FeatureFlagsResponseModel{}
	err := findApplicationAndUpdate(applicationName, applicationWrapper, projectName, projectID, featureFlagsWrapper, tenantWrapper)
	assert.NilError(t, err)
}
