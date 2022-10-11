package utils

import (
	"log"
	"testing"
)

func TestCleanURL(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "cleans correctly",
			args: args{
				uri: "https://codebashing.checkmarx.com/courses/java/////lessons/sql_injection/////",
			},
			want:    "https://codebashing.checkmarx.com/courses/java/lessons/sql_injection",
			wantErr: false,
		},
		{
			name: "invalid URL escape error",
			args: args{
				uri: "#)@($_(*#_(*@$_))%(_#@_+#@$)$_$#@_@_##}^^^}!)(()!#@(`SPPSCOK^Ç^Ç`P$_$",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "cleans correctly",
			args: args{
				uri: "http://localhost:42/////test//test",
			},
			want:    "http://localhost:42/test/test",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CleanURL(tt.args.uri)
			log.Println(err)
			if (err != nil) != tt.wantErr {
				t.Errorf("CleanURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CleanURL() got = %v, want %v", got, tt.want)
			}
		})
	}
}
