package services

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
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
			err := associateProjectToApplication(tt.applicationName, tt.projectName, applicationWrapper)
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

func Test_AssociateProjectToApplication_ProjectAlreadyAssociated(t *testing.T) {
	projectID := "project-123"
	applicationName := "app-1"
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	err := associateProjectToApplication(applicationName, projectID, applicationWrapper)
	assert.NilError(t, err)
}

func resetFeatureFlagState() {
	mock.Flags = nil
	mock.Flag = wrappers.FeatureFlagResponseModel{}
	mock.FFErr = nil //nolint:gocritic // resetting shared mock package state between tests
	mock.TenantConfiguration = nil
	wrappers.ClearCache()
}

func TestGetApplication_EmptyName_ReturnsNilNil(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	application, err := GetApplication("", applicationWrapper)
	assert.NilError(t, err)
	assert.Assert(t, application == nil)
}

func TestGetApplication_NotFound_ReturnsNilNil(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	application, err := GetApplication("anyApplication", applicationWrapper)
	assert.NilError(t, err)
	assert.Assert(t, application == nil)
}

func TestGetApplication_Found_ReturnsApplication(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	application, err := GetApplication("MOCK", applicationWrapper)
	assert.NilError(t, err)
	assert.Assert(t, application != nil)
	assert.Equal(t, application.Name, "MOCK")
}

func TestGetApplication_NoExactNameMatch_ReturnsNil(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	application, err := GetApplication("some-other-application-name", applicationWrapper)
	assert.NilError(t, err)
	assert.Assert(t, application == nil)
}

func TestGetApplication_WrapperError_ReturnsError(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	application, err := GetApplication(mock.NoPermissionApp, applicationWrapper)
	assert.Assert(t, err != nil)
	assert.Assert(t, application == nil)
}

func TestGetApplicationID_EmptyName_ReturnsNilNil(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	ids, err := getApplicationID("", applicationWrapper)
	assert.NilError(t, err)
	assert.Assert(t, ids == nil)
}

func TestGetApplicationID_Found_ReturnsID(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	ids, err := getApplicationID("MOCK", applicationWrapper)
	assert.NilError(t, err)
	assert.DeepEqual(t, ids, []string{"mockID"})
}

func TestGetApplicationID_NotFound_ReturnsError(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	ids, err := getApplicationID("anyApplication", applicationWrapper)
	assert.Assert(t, err != nil)
	assert.Assert(t, ids == nil)
}

func TestGetApplicationID_WrapperError_ReturnsError(t *testing.T) {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	ids, err := getApplicationID(mock.NoPermissionApp, applicationWrapper)
	assert.Assert(t, err != nil)
	assert.Assert(t, ids == nil)
}

func TestCheckDirectAssociationEnabled_DirectFlagEnabled_ReturnsTrue(t *testing.T) {
	resetFeatureFlagState()
	defer resetFeatureFlagState()
	mock.Flags = wrappers.FeatureFlagsResponseModel{
		{Name: wrappers.DirectAssociationEnabled, Status: true},
		{Name: wrappers.DaMigrationEnabled, Status: false},
	}
	enabled, err := checkDirectAssociationEnabled(&mock.FeatureFlagsMockWrapper{}, &mock.TenantConfigurationMockWrapper{})
	assert.NilError(t, err)
	assert.Assert(t, enabled)
}

func TestCheckDirectAssociationEnabled_BothDisabled_ReturnsFalse(t *testing.T) {
	resetFeatureFlagState()
	defer resetFeatureFlagState()
	mock.Flags = wrappers.FeatureFlagsResponseModel{
		{Name: wrappers.DirectAssociationEnabled, Status: false},
		{Name: wrappers.DaMigrationEnabled, Status: false},
	}
	enabled, err := checkDirectAssociationEnabled(&mock.FeatureFlagsMockWrapper{}, &mock.TenantConfigurationMockWrapper{})
	assert.NilError(t, err)
	assert.Assert(t, !enabled)
}

func TestCheckDirectAssociationEnabled_MigrationEnabledWithConfig_ReturnsTrue(t *testing.T) {
	resetFeatureFlagState()
	defer resetFeatureFlagState()
	mock.Flags = wrappers.FeatureFlagsResponseModel{
		{Name: wrappers.DirectAssociationEnabled, Status: false},
		{Name: wrappers.DaMigrationEnabled, Status: true},
	}
	enabled, err := checkDirectAssociationEnabled(&mock.FeatureFlagsMockWrapper{}, &mock.TenantConfigurationMockWrapper{})
	assert.NilError(t, err)
	assert.Assert(t, enabled)
}

func TestCheckDirectAssociationEnabled_MigrationEnabledWrapperError_ReturnsError(t *testing.T) {
	resetFeatureFlagState()
	defer resetFeatureFlagState()
	mock.Flags = wrappers.FeatureFlagsResponseModel{
		{Name: wrappers.DirectAssociationEnabled, Status: false},
		{Name: wrappers.DaMigrationEnabled, Status: true},
	}
	tenantWrapper := &mock.TenantConfigurationMockWrapper{
		CustomGetTenantConfiguration: func() (*[]*wrappers.TenantConfigurationResponse, *wrappers.WebError, error) {
			return nil, nil, errors.New("tenant configuration request failed")
		},
	}
	enabled, err := checkDirectAssociationEnabled(&mock.FeatureFlagsMockWrapper{}, tenantWrapper)
	assert.Assert(t, err != nil)
	assert.Assert(t, !enabled)
}

func TestFindApplicationAndUpdate_EmptyName_ReturnsNil(t *testing.T) {
	err := findApplicationAndUpdate("", &mock.ApplicationsMockWrapper{}, "project-name", "project-id",
		&mock.FeatureFlagsMockWrapper{}, &mock.TenantConfigurationMockWrapper{})
	assert.NilError(t, err)
}

func TestFindApplicationAndUpdate_ApplicationNotFound_ReturnsError(t *testing.T) {
	err := findApplicationAndUpdate("anyApplication", &mock.ApplicationsMockWrapper{}, "project-name", "project-id",
		&mock.FeatureFlagsMockWrapper{}, &mock.TenantConfigurationMockWrapper{})
	assert.Assert(t, err != nil)
}

func TestFindApplicationAndUpdate_GetApplicationError_ReturnsError(t *testing.T) {
	err := findApplicationAndUpdate(mock.NoPermissionApp, &mock.ApplicationsMockWrapper{}, "project-name", "project-id",
		&mock.FeatureFlagsMockWrapper{}, &mock.TenantConfigurationMockWrapper{})
	assert.Assert(t, err != nil)
}

func TestFindApplicationAndUpdate_AlreadyAssociated_ReturnsNil(t *testing.T) {
	err := findApplicationAndUpdate(mock.ExistingApplication, &mock.ApplicationsMockWrapper{}, "project-name", "ID-newProject",
		&mock.FeatureFlagsMockWrapper{}, &mock.TenantConfigurationMockWrapper{})
	assert.NilError(t, err)
}

func TestFindApplicationAndUpdate_DirectAssociationEnabled_AssociatesProject(t *testing.T) {
	resetFeatureFlagState()
	defer resetFeatureFlagState()
	mock.Flags = wrappers.FeatureFlagsResponseModel{
		{Name: wrappers.DirectAssociationEnabled, Status: true},
		{Name: wrappers.DaMigrationEnabled, Status: false},
	}
	err := findApplicationAndUpdate("MOCK", &mock.ApplicationsMockWrapper{}, "project-name", "brand-new-project-id",
		&mock.FeatureFlagsMockWrapper{}, &mock.TenantConfigurationMockWrapper{})
	assert.NilError(t, err)
}

func TestFindApplicationAndUpdate_DirectAssociationDisabled_UpdatesApplication(t *testing.T) {
	resetFeatureFlagState()
	defer resetFeatureFlagState()
	mock.Flags = wrappers.FeatureFlagsResponseModel{
		{Name: wrappers.DirectAssociationEnabled, Status: false},
		{Name: wrappers.DaMigrationEnabled, Status: false},
	}
	err := findApplicationAndUpdate("MOCK", &mock.ApplicationsMockWrapper{}, "project-name", "brand-new-project-id-2",
		&mock.FeatureFlagsMockWrapper{}, &mock.TenantConfigurationMockWrapper{})
	assert.NilError(t, err)
}
