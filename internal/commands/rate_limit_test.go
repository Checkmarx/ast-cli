package commands

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
		// Ensure Body is non-nil to satisfy bodyclose linter
		resp.Body = io.NopCloser(strings.NewReader(""))
		return resp, nil
	}
}

func TestGitHubRateLimit_SuccessAfterRetryOne(t *testing.T) {
	reset := strconv.FormatInt(time.Now().Unix()+20, 10) // simulate 20-second wait
	api := mockAPI(403, 1, "X-RateLimit-Reset", reset)

	start := time.Now()
	resp, err := wrappers.WithSCMRateLimitRetry(wrappers.GitHubRateLimitConfig, api)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		asserts := assert.New(t)
		asserts.NoError(err)
		return
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	asserts := assert.New(t)
	asserts.NoError(err)
	asserts.Equal(http.StatusOK, resp.StatusCode)
	asserts.GreaterOrEqual(elapsed, 20*time.Second)
}

func TestGitHubRateLimit_SuccessAfterRetryTwo(t *testing.T) {
	reset := strconv.FormatInt(time.Now().Unix()+20, 10) // simulate 20-second wait
	api := mockAPI(429, 2, "X-RateLimit-Reset", reset)

	start := time.Now()
	resp, err := wrappers.WithSCMRateLimitRetry(wrappers.GitHubRateLimitConfig, api)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		asserts := assert.New(t)
		asserts.NoError(err)
		return
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	asserts := assert.New(t)
	asserts.NoError(err)
	asserts.Equal(http.StatusOK, resp.StatusCode)
	asserts.GreaterOrEqual(elapsed, 20*time.Second)
}

func TestGitHubRateLimit_SuccessAfterRetryThree(t *testing.T) {
	reset := strconv.FormatInt(time.Now().Unix()+20, 10) // simulate 20-second wait
	api := mockAPI(403, 3, "X-RateLimit-Reset", reset)

	start := time.Now()
	resp, err := wrappers.WithSCMRateLimitRetry(wrappers.GitHubRateLimitConfig, api)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		asserts := assert.New(t)
		asserts.NoError(err)
		return
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	asserts := assert.New(t)
	asserts.NoError(err)
	asserts.Equal(http.StatusOK, resp.StatusCode)
	asserts.GreaterOrEqual(elapsed, 20*time.Second)
}

func TestGitLabRateLimit_SuccessAfterRetryOne(t *testing.T) {
	reset := strconv.FormatInt(time.Now().Unix()+20, 10) // simulate 20-second wait
	api := mockAPI(429, 1, "RateLimit-Reset", reset)

	start := time.Now()
	resp, err := wrappers.WithSCMRateLimitRetry(wrappers.GitLabRateLimitConfig, api)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		asserts := assert.New(t)
		asserts.NoError(err)
		return
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	asserts := assert.New(t)
	asserts.NoError(err)
	asserts.Equal(http.StatusOK, resp.StatusCode)
	asserts.GreaterOrEqual(elapsed, 20*time.Second)
}

func TestBitBucketRateLimit_SuccessAfterRetryOne(t *testing.T) {
	reset := strconv.FormatInt(time.Now().Unix()+20, 10) // simulate 20-second wait
	api := mockAPI(429, 1, "X-RateLimit-Reset", reset)

	start := time.Now()
	resp, err := wrappers.WithSCMRateLimitRetry(wrappers.BitbucketRateLimitConfig, api)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		asserts := assert.New(t)
		asserts.NoError(err)
		return
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	asserts := assert.New(t)
	asserts.NoError(err)
	asserts.Equal(http.StatusOK, resp.StatusCode)
	asserts.GreaterOrEqual(elapsed, 20*time.Second)
}

func TestAzureRateLimit_SuccessAfterRetryOne(t *testing.T) {
	reset := strconv.FormatInt(time.Now().Unix()+20, 10) // simulate 20-second wait
	api := mockAPI(429, 1, "X-Ratelimit-Reset", reset)

	start := time.Now()
	resp, err := wrappers.WithSCMRateLimitRetry(wrappers.AzureRateLimitConfig, api)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		asserts := assert.New(t)
		asserts.NoError(err)
		return
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	asserts := assert.New(t)
	asserts.NoError(err)
	asserts.Equal(http.StatusOK, resp.StatusCode)
	asserts.GreaterOrEqual(elapsed, 20*time.Second)
}
