package services

import (
	"reflect"
	"testing"
)

func Test_createTagMap(t *testing.T) {
	type args struct {
		tagListStr string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		// TODO: Add test cases.
		{
			name: "Create tag map from tag string representing a map",
			args: args{tagListStr: "tag1:val1,tag2:val2"},
			want: map[string]string{"tag1": "val1", "tag2": "val2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createTagMap(tt.args.tagListStr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createTagMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
