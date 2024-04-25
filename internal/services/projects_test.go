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
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "Testing the update flow",
			args: args{
				applicationID:           []string{"1"},
				projectName:             "MOCK",
				cmd:                     &cobra.Command{},
				projectsWrapper:         &mock.ProjectsMockWrapper{},
				groupsWrapper:           &mock.GroupsMockWrapper{},
				accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			},
			want:    "MOCK",
			wantErr: false,
		},
		{
			name: "Testing the create flow",
			args: args{
				applicationID:           []string{"1"},
				projectName:             "new-MOCK",
				cmd:                     &cobra.Command{},
				projectsWrapper:         &mock.ProjectsMockWrapper{},
				groupsWrapper:           &mock.GroupsMockWrapper{},
				accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			},
			want:    "ID-new-MOCK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindProject(ttt.args.applicationID, ttt.args.projectName, ttt.args.cmd, ttt.args.projectsWrapper, ttt.args.groupsWrapper, ttt.args.accessManagementWrapper)
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
		applicationID           []string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "When called with a new project name return the Id of the newly created project", args: args{
			projectName:             "new-project-name",
			cmd:                     &cobra.Command{},
			projectsWrapper:         &mock.ProjectsMockWrapper{},
			groupsWrapper:           &mock.GroupsMockWrapper{},
			accessManagementWrapper: &mock.AccessManagementMockWrapper{},
			applicationID:           []string{"1"},
		}, want: "ID-new-project-name", wantErr: false},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := createProject(ttt.args.projectName, ttt.args.cmd, ttt.args.projectsWrapper, ttt.args.groupsWrapper, ttt.args.accessManagementWrapper, ttt.args.applicationID)
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
		projectName             string
		applicationID           []string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
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
				ttt.args.projectName,
				ttt.args.applicationID)
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
