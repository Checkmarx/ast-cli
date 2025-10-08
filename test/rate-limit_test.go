package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/stretchr/testify/assert"
)

func mockAPI(repeatCode, repeatCount int, headerName, headerValue string) func() (*http.Response, error) {
	attempt := 0
	return func() (*http.Response, error) {
		rec := httptest.NewRecorder()
		if attempt < repeatCount {
			rec.Code = repeatCode
			if headerName != "" {
				rec.Header().Set(headerName, headerValue)
			}
		} else {
			rec.Code = http.StatusOK
		}
		attempt++
		resp := rec.Result()
		resp.Body = io.NopCloser(strings.NewReader(""))
		return resp, nil
	}
}

func runRateLimitTest(t *testing.T, config *wrappers.SCMRateLimitConfig, repeatCode, repeatCount int, headerName string) {
	reset := strconv.FormatInt(time.Now().Unix(), 10) // simulate immediate retry
	api := mockAPI(repeatCode, repeatCount, headerName, reset)

	start := time.Now()
	resp, err := wrappers.WithSCMRateLimitRetry(config, api)
	if resp != nil {
		defer resp.Body.Close()
	}

	assert := assert.New(t)
	assert.NoError(err)
	assert.NotNil(resp)
	assert.Equal(http.StatusOK, resp.StatusCode)

	elapsed := time.Since(start)
	assert.GreaterOrEqual(elapsed, config.DefaultWaitTime)
}

func TestGitHubRateLimit_SuccessAfterRetryOne(t *testing.T) {
	runRateLimitTest(t, wrappers.GitHubRateLimitConfig, 429, 1, "X-RateLimit-Reset")
}

func TestGitHubRateLimit_SuccessAfterRetryTwo(t *testing.T) {
	runRateLimitTest(t, wrappers.GitHubRateLimitConfig, 429, 2, "X-RateLimit-Reset")
}

func TestGitHubRateLimit_SuccessAfterRetryThree(t *testing.T) {
	runRateLimitTest(t, wrappers.GitHubRateLimitConfig, 403, 3, "X-RateLimit-Reset")
}

func TestGitLabRateLimit_SuccessAfterRetryOne(t *testing.T) {
	runRateLimitTest(t, wrappers.GitLabRateLimitConfig, 429, 1, "RateLimit-Reset")
}

func TestBitBucketRateLimit_SuccessAfterRetryOne(t *testing.T) {
	runRateLimitTest(t, wrappers.BitbucketRateLimitConfig, 429, 1, "X-RateLimit-Reset")
}

func TestAzureRateLimit_SuccessAfterRetryOne(t *testing.T) {
	runRateLimitTest(t, wrappers.AzureRateLimitConfig, 429, 1, "X-Ratelimit-Reset")
}
