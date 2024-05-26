//go:build !integration

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
