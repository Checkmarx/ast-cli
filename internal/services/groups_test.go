package services

import (
	"reflect"
	"testing"

	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

func TestAssignGroupsToProject(t *testing.T) {
	type args struct {
		projectID        string
		projectName      string
		groups           []*wrappers.Group
		accessManagement wrappers.AccessManagementWrapper
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "When assigning group to project, no error should be returned",
			args: args{
				projectID:   "project-id",
				projectName: "project-name",
				groups: []*wrappers.Group{{
					ID:   "group-id-to-assign",
					Name: "group-name-to-assign",
				}},
				accessManagement: &mock.AccessManagementMockWrapper{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		wrappers.FeatureFlags[featureFlagsConstants.AccessManagementEnabled] = true
		t.Run(tt.name, func(t *testing.T) {
			if err := AssignGroupsToProject(tt.args.projectID, tt.args.projectName, tt.args.groups, tt.args.accessManagement); (err != nil) != tt.wantErr {
				t.Errorf("AssignGroupsToProject() error = %v, wantErr %v", err, tt.wantErr)
			}
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
		// TODO: Add test cases.
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateGroupsMap(tt.args.groupsStr, tt.args.groupsWrapper)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateGroupsMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateGroupsMap() got = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
		{
			name: "When passing a slice of groups, return a slice of group IDs",
			args: args{groups: []*wrappers.Group{{ID: "group-id-1", Name: "group-name-1"}, {ID: "group-id-2", Name: "group-name-2"}}},
			want: []string{"group-id-1", "group-id-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetGroupIds(tt.args.groups); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGroupIds() = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
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
		t.Run(tt.name, func(t *testing.T) {
			if got := findGroupByName(tt.args.groups, tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findGroupByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getGroupsForRequest(t *testing.T) {
	type args struct {
		groups []*wrappers.Group
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
		{
			name: "When access management is disabled, return group IDs of the groups",
			args: args{groups: []*wrappers.Group{{ID: "group-id-1", Name: "group-name-1"}, {ID: "group-id-2", Name: "group-name-2"}}},
			want: []string{"group-id-1", "group-id-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappers.FeatureFlags[featureFlagsConstants.AccessManagementEnabled] = false
			if got := getGroupsForRequest(tt.args.groups); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getGroupsForRequest() = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
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
		t.Run(tt.name, func(t *testing.T) {
			if got := getGroupsToAssign(tt.args.receivedGroups, tt.args.existingGroups); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getGroupsToAssign() = %v, want %v", got, tt.want)
			}
		})
	}
}
