package services

import (
	"bytes"
	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

func setup() {
	wrappers.ClearCache()

}

func TestAssignGroupsToProject(t *testing.T) {
	setup() // Clear the map before starting this test
	type args struct {
		projectID           string
		projectName         string
		groups              []*wrappers.Group
		accessManagement    wrappers.AccessManagementWrapper
		featureFlagsWrapper wrappers.FeatureFlagsWrapper
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "When assigning group to project, no error should be returned when AM1 is ON",
			args: args{
				projectID:   "project-id",
				projectName: "project-name",
				groups: []*wrappers.Group{{
					ID:   "group-id-to-assign",
					Name: "group-name-to-assign",
				}},
				accessManagement:    &mock.AccessManagementMockWrapper{},
				featureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			mock.Flag = wrappers.FeatureFlagResponseModel{Name: featureFlagsConstants.AccessManagementEnabled, Status: true}
			mock.Flag = wrappers.FeatureFlagResponseModel{Name: featureFlagsConstants.GroupValidationEnabled, Status: false}
			var output string
			output = captureStdout(func() {
				if err := AssignGroupsToProjectNewAccessManagement(ttt.args.projectID, ttt.args.projectName, ttt.args.groups,
					ttt.args.accessManagement, ttt.args.featureFlagsWrapper); (err != nil) != ttt.wantErr {
					t.Errorf("AssignGroupsToProjectNewAccessManagement() error = %v, wantErr %v", err, ttt.wantErr)
					if err != nil {
						t.Errorf("failed to close file")
					}

					if output != "" && !strings.Contains(output, "Called CreateGroupsAssignment in AccessManagementMockWrapper") {
						t.Errorf("Create GroupAssignment should not get called when GroupValidation Flag is ON")
					}

				}
			})

		})
	}
}

func TestAssignGroupsToProjectWithGroupValidationFlag(t *testing.T) {
	setup() // Clear the map before starting this test
	type args struct {
		projectID           string
		projectName         string
		groups              []*wrappers.Group
		accessManagement    wrappers.AccessManagementWrapper
		featureFlagsWrapper wrappers.FeatureFlagsWrapper
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "When assigning group to project,error should be returned when Access_management_enabled is ON and GroupValidationFlag is also On",
			args: args{
				projectID:   "project-id",
				projectName: "project-name",
				groups: []*wrappers.Group{{
					ID:   "group-id-to-assign",
					Name: "group-name-to-assign",
				}},
				accessManagement:    &mock.AccessManagementMockWrapper{},
				featureFlagsWrapper: &mock.FeatureFlagsMockWrapper{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			mock.Flag = wrappers.FeatureFlagResponseModel{Name: featureFlagsConstants.AccessManagementEnabled, Status: true}
			mock.Flag = wrappers.FeatureFlagResponseModel{Name: featureFlagsConstants.GroupValidationEnabled, Status: true}
			var output string
			output = captureStdout(func() {
				if err := AssignGroupsToProjectNewAccessManagement(ttt.args.projectID, ttt.args.projectName, ttt.args.groups,
					ttt.args.accessManagement, ttt.args.featureFlagsWrapper); (err != nil) != ttt.wantErr {
					t.Errorf("AssignGroupsToProjectNewAccessManagement() error = %v, wantErr %v", err, ttt.wantErr)
					if err != nil {
						t.Errorf("failed to close file")
					}
					if output != "" && strings.Contains(output, "Called CreateGroupsAssignment in AccessManagementMockWrapper") {
						t.Errorf("Create GroupAssignment should not get called when GroupValidation Flag is ON")
					}
				}
			})

		})
	}
}

func TestCreateGroupsMap(t *testing.T) {
	type args struct {
		groupsStr     string
		groupsWrapper wrappers.GroupsWrapper
	}
	tests := []struct {
		name    string
		args    args
		want    []*wrappers.Group
		wantErr bool
	}{
		{
			name: "When creating a group map with existing group, no error should be returned",
			args: args{
				groupsStr:     "group",
				groupsWrapper: &mock.GroupsMockWrapper{},
			},
			want:    []*wrappers.Group{{ID: "1", Name: "group"}},
			wantErr: false,
		},
		{
			name: "When creating a group map with non-existing group, an error should be returned",
			args: args{
				groupsStr:     "not-existing-group",
				groupsWrapper: &mock.GroupsMockWrapper{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "When faking an error upon calling groups wrapper, an error should be returned",
			args: args{
				groupsStr:     "fake-group-error",
				groupsWrapper: &mock.GroupsMockWrapper{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateGroupsMap(ttt.args.groupsStr, ttt.args.groupsWrapper)
			if (err != nil) != ttt.wantErr {
				t.Errorf("CreateGroupsMap() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, ttt.want) {
				t.Errorf("CreateGroupsMap() got = %v, want %v", got, ttt.want)
			}
		})
	}
}

func TestGetGroupIds(t *testing.T) {
	type args struct {
		groups []*wrappers.Group
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "When passing a slice of groups, return a slice of group IDs",
			args: args{groups: []*wrappers.Group{{ID: "group-id-1", Name: "group-name-1"}, {ID: "group-id-2", Name: "group-name-2"}}},
			want: []string{"group-id-1", "group-id-2"},
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := GetGroupIds(ttt.args.groups); !reflect.DeepEqual(got, ttt.want) {
				t.Errorf("GetGroupIds() = %v, want %v", got, ttt.want)
			}
		})
	}
}

func Test_findGroupByName(t *testing.T) {
	type args struct {
		groups []wrappers.Group
		name   string
	}
	tests := []struct {
		name string
		args args
		want *wrappers.Group
	}{
		{
			name: "When calling with a group name, return the group with the same name",
			args: args{
				groups: []wrappers.Group{{
					ID:   "1",
					Name: "group-one",
				},
					{
						ID:   "2",
						Name: "group-two",
					}},
				name: "group-two",
			},
			want: &wrappers.Group{
				ID:   "2",
				Name: "group-two",
			},
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := findGroupByName(ttt.args.groups, ttt.args.name); !reflect.DeepEqual(got, ttt.want) {
				t.Errorf("findGroupByName() = %v, want %v", got, ttt.want)
			}
		})
	}
}

func Test_getGroupsToAssign(t *testing.T) {
	type args struct {
		receivedGroups []*wrappers.Group
		existingGroups []*wrappers.Group
	}
	tests := []struct {
		name string
		args args
		want []*wrappers.Group
	}{
		{
			name: "When calling with received groups, assign only the non-existing ones",
			args: args{
				receivedGroups: []*wrappers.Group{{ID: "group-id-2", Name: "group-name-2"}, {ID: "group-id-3", Name: "group-name-3"}},
				existingGroups: []*wrappers.Group{{ID: "group-id-1", Name: "group-name-1"}, {ID: "group-id-2", Name: "group-name-2"}},
			},
			want: []*wrappers.Group{{ID: "group-id-3", Name: "group-name-3"}},
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := getGroupsToAssign(ttt.args.receivedGroups, ttt.args.existingGroups); !reflect.DeepEqual(got, ttt.want) {
				t.Errorf("getGroupsToAssign() = %v, want %v", got, ttt.want)
			}
		})
	}
}

func captureStdout(fn func()) string {
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = originalStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)

	return buf.String()
}
