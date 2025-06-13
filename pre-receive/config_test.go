package pre_receive

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigExcludesToGitExcludes(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"empty", []string{}, nil},
		{"single filename", []string{"a.txt"}, []string{":(exclude)a.txt"}},
		{"single path", []string{"dir/a.txt"}, []string{":(exclude)dir/a.txt"}},
		{"windows backslash", []string{`"\folder\file.txt"`}, []string{":(exclude)folder/file.txt"}},
		{"leading slash", []string{"/root.txt"}, []string{":(exclude)root.txt"}},
		{"multiple patterns", []string{"a.txt", "dir/b.log"}, []string{":(exclude)a.txt", ":(exclude)dir/b.log"}},
		{"trims spaces", []string{" a.txt", "dir/c.md "}, []string{":(exclude)a.txt", ":(exclude)dir/c.md"}},
		{"filename with space", []string{"my file.txt"}, []string{":(exclude)my file.txt"}},
		{"path with spaces", []string{"dir name/file name.txt"}, []string{":(exclude)dir name/file name.txt"}},

		// Glob‐pattern cases
		{"simple glob", []string{"*.log"}, []string{":(exclude)*.log"}},
		{"glob path", []string{"dir/*.js"}, []string{":(exclude)dir/*.js"}},
		{"recursive glob", []string{"src/**/*.go"}, []string{":(exclude)src/**/*.go"}},
		{"glob with quotes", []string{`"*.tmp"`}, []string{":(exclude)*.tmp"}},
		{"leading slash glob", []string{"/cache/*.cache"}, []string{":(exclude)cache/*.cache"}},
		{"trimmed glob", []string{" *.md "}, []string{":(exclude)*.md"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := configExcludesToGitExcludes(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
