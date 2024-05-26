//go:build !integration

package services

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
)

func TestFindProject(t *testing.T) {
	type args struct {
		applicationID           []string
		projectName             string
		cmd                     *cobra.Command
		projectsWrapper         wrappers.ProjectsWrapper
		groupsWrapper           wrappers.GroupsWrapper
		accessManagementWrapper wrappers.AccessManagementWrapper
		applicationsWrapper     wrappers.ApplicationsWrapper
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Testing the update flow",
			args: args{
				applicationID:           []string{"1"},
				projectName:             "MOCK",
				cmd:                     &cobra.Command{},
				projectsWrapper:         &mock.ProjectsMockWrapper{},
				groupsWrapper:           &mock.GroupsMockWrapper{},
				accessManagementWrapper: &mock.AccessManagementMockWrapper{},
				applicationsWrapper:     &mock.ApplicationsMockWrapper{},
			},
			want:    "MOCK",
			wantErr: false,
		},
		{
			name: "Testing the create flow",
			args: args{
				projectName:             "new-MOCK",
				cmd:                     &cobra.Command{},
				projectsWrapper:         &mock.ProjectsMockWrapper{},
				groupsWrapper:           &mock.GroupsMockWrapper{},
				accessManagementWrapper: &mock.AccessManagementMockWrapper{},
				applicationsWrapper:     &mock.ApplicationsMockWrapper{},
			},
			want:    "ID-new-MOCK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindProject(
				ttt.args.applicationID,
				ttt.args.projectName,
				ttt.args.cmd,
				ttt.args.projectsWrapper,
				ttt.args.groupsWrapper,
				ttt.args.accessManagementWrapper,
				ttt.args.applicationsWrapper)
			if (err != nil) != ttt.wantErr {
				t.Errorf("FindProject() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if got != ttt.want {
				t.Errorf("FindProject() got = %v, want %v", got, ttt.want)
			}
		})
	}
}

func Test_createProject(t *testing.T) {
	type args struct {
		projectName             string
		cmd                     *cobra.Command
		projectsWrapper         wrappers.ProjectsWrapper
		groupsWrapper           wrappers.GroupsWrapper
		accessManagementWrapper wrappers.AccessManagementWrapper
		applicationsWrapper     wrappers.ApplicationsWrapper
		applicationID           []string
		projectGroups           string
		projectPrivatePackage   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "When called with a new project name return the Id of the newly created project", args: args{
			projectName:             "new-project-name",
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			projectGroups:           "",
		}, want: "ID-new-project-name", wantErr: false},
		{name: "When called with a new project name and non existing project groups return error", args: args{
			projectName:             "new-project-name",
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			projectGroups:           "grp1,grp2",
		}, want: "ID-new-project-name", wantErr: true},
		{name: "When called with mock fake error model return fake error from project create", args: args{
			projectName:             "mock-some-error-model",
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			projectGroups:           "",
		}, want: "", wantErr: true},
		{name: "When called with mock fake group error return fake error from project create", args: args{
			projectName:             "new-project-name",
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			projectGroups:           "fake-group-error",
		}, want: "ID-new-project-name", wantErr: true},
		{name: "When called with a new project name and projectPrivatePackage set to true return the Id of the newly created project", args: args{
			projectName:             "new-project-name",
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			projectGroups:           "",
			projectPrivatePackage:   "true",
		}, want: "ID-new-project-name", wantErr: false},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := createProject(
				ttt.args.projectName,
				ttt.args.cmd,
				ttt.args.projectsWrapper,
				ttt.args.groupsWrapper,
				ttt.args.accessManagementWrapper,
				ttt.args.applicationsWrapper,
				ttt.args.applicationID,
				ttt.args.projectGroups,
				ttt.args.projectPrivatePackage)
			if (err != nil) != ttt.wantErr {
				t.Errorf("createProject() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if got != ttt.want {
				t.Errorf("createProject() got = %v, want %v", got, ttt.want)
			}
		})
	}
}

func Test_updateProject(t *testing.T) {
	type args struct {
		resp                    *wrappers.ProjectsCollectionResponseModel
		cmd                     *cobra.Command
		projectsWrapper         wrappers.ProjectsWrapper
		groupsWrapper           wrappers.GroupsWrapper
		accessManagementWrapper wrappers.AccessManagementWrapper
		applicationsWrapper     wrappers.ApplicationsWrapper
		projectName             string
		applicationID           []string
		projectGroups           string
		projectTags             string
		projectPrivatePackage   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "When called with existing project, update the project and return the project Id", args: args{
			resp: &wrappers.ProjectsCollectionResponseModel{
				Projects: []wrappers.ProjectResponseModel{
					{ID: "ID-project-name", Name: "project-name"}},
			},
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			projectName:             "project-name",
			applicationID:           nil,
		}, want: "ID-project-name", wantErr: false},
		{name: "without app ID and with project tags", args: args{
			resp: &wrappers.ProjectsCollectionResponseModel{
				Projects: []wrappers.ProjectResponseModel{
					{ID: "ID-project-name", Name: "project-name"}},
			},
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			projectName:             "project-name",
			projectTags:             "tag1,tag2",
			applicationID:           nil,
		}, want: "ID-project-name", wantErr: false},
		{name: "When called with application ID", args: args{
			resp: &wrappers.ProjectsCollectionResponseModel{
				Projects: []wrappers.ProjectResponseModel{
					{ID: "ID-project-name", Name: "project-name"}},
			},
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			projectName:             "project-name",
			projectPrivatePackage:   "true",
		}, want: "ID-project-name", wantErr: false},
		{name: "When called with mock fake error model return fake error from project create", args: args{
			projectName: "mock-some-error-model",
			resp: &wrappers.ProjectsCollectionResponseModel{
				Projects: []wrappers.ProjectResponseModel{
					{ID: "ID-mock-some-error-model", Name: "mock-some-error-model"}},
			},
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			applicationID:           []string{"1"},
		}, want: "", wantErr: true},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateProject(
				ttt.args.resp,
				ttt.args.cmd,
				ttt.args.projectsWrapper,
				ttt.args.groupsWrapper,
				ttt.args.accessManagementWrapper,
				ttt.args.applicationsWrapper,
				ttt.args.projectName,
				ttt.args.applicationID,
				ttt.args.projectGroups,
				ttt.args.projectTags,
				ttt.args.projectPrivatePackage)
			if (err != nil) != ttt.wantErr {
				t.Errorf("updateProject() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if got != ttt.want {
				t.Errorf("updateProject() got = %v, want %v", got, ttt.want)
			}
		})
	}
}
