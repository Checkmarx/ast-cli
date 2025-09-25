package usercount

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

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
	GitHubRateLimitConfig = SCMRateLimitConfig{
		Provider:             "GitHub",
		ResetHeaderName:      "X-RateLimit-Reset",
		RemainingHeaderName:  "X-RateLimit-Remaining",
		LimitHeaderName:      "X-RateLimit-Limit",
		RateLimitStatusCodes: []int{403, 429},
		DefaultWaitTime:      60 * time.Second,
	}

	GitLabRateLimitConfig = SCMRateLimitConfig{
		Provider:             "GitLab",
		ResetHeaderName:      "RateLimit-Reset",
		RemainingHeaderName:  "RateLimit-Remaining",
		LimitHeaderName:      "RateLimit-Limit",
		RateLimitStatusCodes: []int{429},
		DefaultWaitTime:      60 * time.Second,
	}

	BitbucketRateLimitConfig = SCMRateLimitConfig{
		Provider:             "Bitbucket",
		ResetHeaderName:      "X-RateLimit-Reset",
		RemainingHeaderName:  "X-RateLimit-Remaining",
		LimitHeaderName:      "X-RateLimit-Limit",
		RateLimitStatusCodes: []int{429},
		DefaultWaitTime:      60 * time.Second,
	}

	AzureRateLimitConfig = SCMRateLimitConfig{
		Provider:             "Azure",
		ResetHeaderName:      "X-RateLimit-Reset",
		RemainingHeaderName:  "X-RateLimit-Remaining",
		LimitHeaderName:      "X-RateLimit-Limit",
		RateLimitStatusCodes: []int{429},
		DefaultWaitTime:      60 * time.Second,
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
			return reset.Sub(now) + time.Second
		}
	}
	return 60 * time.Second // Default fallback
}

// WithSCMRateLimitRetry wraps any SCM API call with rate limit retry logic
func WithSCMRateLimitRetry(config SCMRateLimitConfig, apiCall func() error) error {
	maxRetries := 3
	retryCount := 0

	for {
		err := apiCall()
		if err == nil {
			return nil
		}

		// Check if it's a rate limit error
		var rateLimitErr *SCMRateLimitError
		if errors.As(err, &rateLimitErr) {
			if retryCount >= maxRetries {
				return errors.Errorf("%s API rate limit exceeded after %d retries", config.Provider, maxRetries)
			}

			wait := rateLimitErr.RetryAfter()
			log.Printf("%s API rate limit exceeded. Waiting %v before retrying... (attempt %d/%d)",
				config.Provider, wait, retryCount+1, maxRetries)
			time.Sleep(wait)
			retryCount++
			continue
		}

		// Check for generic rate limit error messages
		if isGenericRateLimitError(err, config) {
			if retryCount >= maxRetries {
				return errors.Errorf("%s API rate limit exceeded after %d retries", config.Provider, maxRetries)
			}

			wait := config.DefaultWaitTime
			log.Printf("%s API rate limit exceeded (fallback). Waiting %v before retrying... (attempt %d/%d)",
				config.Provider, wait, retryCount+1, maxRetries)
			time.Sleep(wait)
			retryCount++
			continue
		}

		return err
	}
}

// isGenericRateLimitError checks for generic rate limit error messages
func isGenericRateLimitError(err error, config SCMRateLimitConfig) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	rateLimitKeywords := []string{
		"rate limit exceeded",
		"too many requests",
		"api rate limit",
		"exceeded a secondary rate limit",
		"throttled",
		"rate limited",
	}

	for _, keyword := range rateLimitKeywords {
		if strings.Contains(msg, keyword) {
			return true
		}
	}

	return false
}

// ParseRateLimitHeaders extracts rate limit information from HTTP response headers
func ParseRateLimitHeaders(headers map[string][]string, config SCMRateLimitConfig) *SCMRateLimitError {
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
