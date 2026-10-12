package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeGitHubURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "https", raw: "https://github.com/acme/widgets.git", want: "https://github.com/acme/widgets"},
		{name: "https no suffix", raw: "https://github.com/acme/widgets", want: "https://github.com/acme/widgets"},
		{name: "www", raw: "https://www.github.com/acme/widgets/", want: "https://github.com/acme/widgets"},
		{name: "credentials stripped", raw: "https://user:token@github.com/acme/widgets.git", want: "https://github.com/acme/widgets"},
		{name: "ssh scp", raw: "git@github.com:acme/widgets.git", want: "https://github.com/acme/widgets"},
		{name: "ssh url", raw: "ssh://git@github.com/acme/widgets.git", want: "https://github.com/acme/widgets"},
		{name: "git protocol", raw: "git://github.com/acme/widgets.git", want: "https://github.com/acme/widgets"},
		{name: "tree path ignored", raw: "https://github.com/acme/widgets/tree/main", want: "https://github.com/acme/widgets"},
		{name: "gitlab", raw: "https://gitlab.com/acme/widgets.git", want: ""},
		{name: "owner only", raw: "https://github.com/acme", want: ""},
		{name: "empty", raw: "  ", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeGitHubURL(tt.raw))
		})
	}
}

func TestResolveGitHubURL_ExplicitWins(t *testing.T) {
	assert.Equal(t, "https://github.com/acme/widgets", ResolveGitHubURL("git@github.com:acme/widgets.git", ""))
	assert.Equal(t, "https://github.example.com/acme/widgets", ResolveGitHubURL(" https://github.example.com/acme/widgets ", t.TempDir()))
}

func TestGitHubRemoteURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	git("init")
	git("remote", "add", "origin", "git@github.com:acme/widgets.git")

	assert.Equal(t, "https://github.com/acme/widgets", GitHubRemoteURL(dir))

	sub := filepath.Join(dir, "nested")
	require.NoError(t, os.Mkdir(sub, 0o750))
	assert.Equal(t, "https://github.com/acme/widgets", GitHubRemoteURL(sub))

	assert.Equal(t, "", GitHubRemoteURL(t.TempDir()))
}

func TestGitHubRemoteURL_EmptyDirUsesCwd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/acme/from-cwd.git")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	t.Chdir(dir)
	assert.Equal(t, "https://github.com/acme/from-cwd", GitHubRemoteURL(""))
}
