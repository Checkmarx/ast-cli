package services

import (
	"testing"

	testifyAssert "github.com/stretchr/testify/assert" //nolint:depguard
)

// func TestMicroSastWrapper_Scan(t *testing.T) {
//	port := 8080
//	client := NewMicroSastService(port)
//	scanResult, err := client.Scan("/Users/benalvo/CxDev/workspace/Pheonix-workspace/CxCodeProbe/testdata/Java/samples/Cookies.java")
//	assert.NotNil(t, scanResult)
//	assert.Nil(t, err)
// }

func TestReplaceCurlyApostrophe(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Single curly apostrophe",
			args: args{data: []byte("Here’s an example.")},
			want: []byte("Here's an example."),
		},
		{
			name: "No curly apostrophe",
			args: args{data: []byte("No curly apostrophe here.")},
			want: []byte("No curly apostrophe here."),
		},
		{
			name: "Empty string",
			args: args{data: []byte("")},
			want: []byte(nil),
		},
		{
			name: "Multiple curly apostrophes",
			args: args{data: []byte("It’s John’s book.")},
			want: []byte("It's John's book."),
		},
		{
			name: "Mixed content",
			args: args{data: []byte("Curly: ’, straight: '.")}, // This tests a mix of curly and straight apostrophes
			want: []byte("Curly: ', straight: '."),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			testifyAssert.Equalf(t, tt.want, replaceCurlyApostrophe(tt.args.data), "ReplaceCurlyApostrophe(%v)", tt.args.data)
		})
	}
}

func TestReplaceEnDashWithHyphen(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Single en dash",
			args: args{data: []byte("This is an en dash – test.")},
			want: []byte("This is an en dash - test."),
		},
		{
			name: "Multiple en dashes",
			args: args{data: []byte("En dash – test – with multiple – en dashes.")},
			want: []byte("En dash - test - with multiple - en dashes."),
		},
		{
			name: "No en dash",
			args: args{data: []byte("No en dash here.")},
			want: []byte("No en dash here."),
		},
		{
			name: "Empty string",
			args: args{data: []byte("")},
			want: []byte(nil),
		},
		{
			name: "Mixed content",
			args: args{data: []byte("Mix of characters: – and - and –.")},
			want: []byte("Mix of characters: - and - and -."),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			testifyAssert.Equalf(t, tt.want, replaceEnDashWithHyphen(tt.args.data), "ReplaceEnDashWithHyphen(%v)", tt.args.data)
		})
	}
}

// func TestCheckHealth(t *testing.T) {
//	port := 8080
//	client := NewMicroSastService(port)
//
//	start := time.Now()
//	err := client.CheckHealth()
//	duration := time.Since(start)
//
//	assert.Nil(t, err)
//	assert.Less(t, duration.Milliseconds(), int64(50), "CheckHealth took too long")
// }
