package usercount

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	asserts "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGitHubUserCountOrgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + OrgsFlag,
			"a,b",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil || strings.Contains(err.Error(), "rate"))
}
func TestGitHubUserCountRepos(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + OrgsFlag,
			"a",
			"--" + ReposFlag,
			"a,b,c",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil || strings.Contains(err.Error(), "rate"))
}

func TestGitHubUserCountMissingArgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingArgs)
}

func TestGitHubUserCountMissingOrgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + ReposFlag,
			"repo",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingOrg)
}

func TestGitHubUserCountManyOrgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + ReposFlag,
			"repo",
			"--" + OrgsFlag,
			"a,b,c",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, tooManyOrgs)
}

func TestHandleRateLimit_WaitsAndRetries(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	resp.Header.Set("X-RateLimit-Remaining", "0")
	resetTime := time.Now().Add(50 * time.Second).Unix()
	resp.Header.Set("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", http.NoBody)

	start := time.Now()
	outResp, err := wrappers.HandleRateLimit(resp, client, req, "http://example.com", "token", map[string]string{})
	defer func() {
		if err := outResp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	elapsed := time.Since(start)

	asserts.NoError(t, err)
	asserts.GreaterOrEqual(t, elapsed, 20*time.Second)
}

func TestHandleRateLimit_NoRateLimit(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", http.NoBody)
	outResp, err := wrappers.HandleRateLimit(resp, client, req, "http://example.com", "token", map[string]string{})
	defer func() {
		if err := outResp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	asserts.NoError(t, err)
	assert.Equal(t, resp, outResp)
}
