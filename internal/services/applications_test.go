package services

import (
	"reflect"
	"testing"
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
		// TODO: Add test cases.
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
		t.Run(tt.name, func(t *testing.T) {
			if got := createApplicationIds(tt.args.applicationID, tt.args.existingApplicationIds); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createApplicationIds() = %v, want %v", got, tt.want)
			}
		})
	}
}
