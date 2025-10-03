package wrappers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const defaultRateLimitWaitSeconds = 60

// SCMRateLimitConfig holds rate limit configuration for different SCM providers
type SCMRateLimitConfig struct {
	Provider             string
	ResetHeaderName      string
	RemainingHeaderName  string
	LimitHeaderName      string
	RateLimitStatusCodes []int
	DefaultWaitTime      time.Duration
}

// Common SCM rate limit configurations
var (
	GitHubRateLimitConfig = &SCMRateLimitConfig{
		Provider:             "GitHub",
		ResetHeaderName:      "X-RateLimit-Reset",
		RemainingHeaderName:  "X-RateLimit-Remaining",
		LimitHeaderName:      "X-RateLimit-Limit",
		RateLimitStatusCodes: []int{403, 429},
		DefaultWaitTime:      defaultRateLimitWaitSeconds * time.Second,
	}

	GitLabRateLimitConfig = &SCMRateLimitConfig{
		Provider:             "GitLab",
		ResetHeaderName:      "RateLimit-Reset",
		RemainingHeaderName:  "RateLimit-Remaining",
		LimitHeaderName:      "RateLimit-Limit",
		RateLimitStatusCodes: []int{429},
		DefaultWaitTime:      defaultRateLimitWaitSeconds * time.Second,
	}

	BitbucketRateLimitConfig = &SCMRateLimitConfig{
		Provider:             "Bitbucket",
		ResetHeaderName:      "X-RateLimit-Reset",
		RemainingHeaderName:  "X-RateLimit-Remaining",
		LimitHeaderName:      "X-RateLimit-Limit",
		RateLimitStatusCodes: []int{429},
		DefaultWaitTime:      defaultRateLimitWaitSeconds * time.Second,
	}

	AzureRateLimitConfig = &SCMRateLimitConfig{
		Provider:             "Azure",
		ResetHeaderName:      "X-Ratelimit-Reset",
		RemainingHeaderName:  "X-Ratelimit-Remaining",
		LimitHeaderName:      "X-Ratelimit-Limit",
		RateLimitStatusCodes: []int{429},
		DefaultWaitTime:      defaultRateLimitWaitSeconds * time.Second,
	}
)

// SCMRateLimitError represents a rate limit error from any SCM provider
type SCMRateLimitError struct {
	Provider  string
	ResetTime int64
	Message   string
}

func (e *SCMRateLimitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Provider + " API rate limit exceeded"
}

func (e *SCMRateLimitError) RetryAfter() time.Duration {
	if e.ResetTime > 0 {
		reset := time.Unix(e.ResetTime, 0)
		now := time.Now()
		if reset.After(now) {
			return reset.Sub(now) + (defaultRateLimitWaitSeconds * time.Second) // add buffer for 60 seconds
		}
	}
	return defaultRateLimitWaitSeconds * time.Second
}

// WithSCMRateLimitRetry wraps any SCM API call with rate limit retry logic
func WithSCMRateLimitRetry(config *SCMRateLimitConfig, apiCall func() (*http.Response, error)) (*http.Response, error) {
	maxRetries := 3
	retryCount := 0

	for {
		resp, err := apiCall()
		if err != nil {
			return nil, err
		}

		// Check if it's a rate limit error
		if isRateLimitStatusCode(resp.StatusCode, config) {
			rateLimitErr := ParseRateLimitHeaders(resp.Header, config)
			wait := config.DefaultWaitTime
			if rateLimitErr != nil {
				wait = rateLimitErr.RetryAfter()
			}
			if retryCount >= maxRetries {
				return nil, errors.Errorf("%s API rate limit exceeded after %d retries", config.Provider, maxRetries)
			}
			log.Printf("%s API rate limit exceeded (status %d). Waiting %v until %v before retrying... (attempt %d/%d)",
				config.Provider, resp.StatusCode, wait, time.Now().Add(wait), retryCount+1, maxRetries)
			time.Sleep(wait)
			// Reset Authorization header before retry
			if resp.Request != nil {
				resetAuthorizationHeader(resp.Request)
			}
			retryCount++
			continue
		}
		return resp, err
	}
}

// ParseRateLimitHeaders extracts rate limit information from HTTP response headers
func ParseRateLimitHeaders(headers map[string][]string, config *SCMRateLimitConfig) *SCMRateLimitError {
	resetHeader := getHeaderValue(headers, config.ResetHeaderName)
	if resetHeader == "" {
		return nil
	}

	resetTime, err := strconv.ParseInt(resetHeader, 10, 64)
	if err != nil {
		return nil
	}

	return &SCMRateLimitError{
		Provider:  config.Provider,
		ResetTime: resetTime,
	}
}

// getHeaderValue retrieves a header value in a case-insensitive manner
func getHeaderValue(headers map[string][]string, headerName string) string {
	for name, values := range headers {
		if strings.EqualFold(name, headerName) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// isRateLimitStatusCode checks if the status code indicates a rate limit error
func isRateLimitStatusCode(statusCode int, config *SCMRateLimitConfig) bool {
	for _, code := range config.RateLimitStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

// resetAuthorizationHeader removes the Authorization header from the request
func resetAuthorizationHeader(req *http.Request) {
	req.Header.Del("Authorization")
}
