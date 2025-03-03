package wrappers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type mockReadCloser struct{}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (m *mockReadCloser) Close() error {
	return nil
}

func TestRetryHTTPRequest_Success(t *testing.T) {
	fn := func() (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockReadCloser{},
		}, nil
	}

	resp, err := retryHTTPRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRetryHTTPRequest_RetryOnBadGateway(t *testing.T) {
	attempts := 0
	fn := func() (*http.Response, error) {
		attempts++
		if attempts < retryAttempts {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       &mockReadCloser{},
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockReadCloser{},
		}, nil
	}

	resp, err := retryHTTPRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, retryAttempts, attempts)
}

func TestRetryHTTPRequest_Fail(t *testing.T) {
	fn := func() (*http.Response, error) {
		return nil, errors.New("network error")
	}

	resp, err := retryHTTPRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRetryHTTPRequest_EndWithBadGateway(t *testing.T) {
	fn := func() (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       &mockReadCloser{},
		}, nil
	}

	resp, err := retryHTTPRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestConcurrentWriteCredentialsToCache(t *testing.T) {
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			writeCredentialsToCache(fmt.Sprintf("testToken_%d", i))
		}(i)
	}
	wg.Wait()

	token := viper.Get(commonParams.AstToken)
	assert.NotNil(t, token, "Token should not be nil")

	tokenStr, ok := token.(string)
	assert.True(t, ok, "Token should be a string")

	splitToken := strings.Split(tokenStr, "_")
	assert.Equal(t, 2, len(splitToken), "Token should split into 2 parts")
	assert.Equal(t, "testToken", splitToken[0], "Token prefix should be 'testToken'")

	testTokenNumber, err := strconv.Atoi(splitToken[1])
	assert.NoError(t, err, "The token suffix should be a valid number")
	assert.True(t, testTokenNumber >= 0 && testTokenNumber < 1000,
		"The token number should be within the expected range")
}

func TestExtractAZPFromToken(t *testing.T) {
	// Test cases
	tests := []struct {
		name     string
		token    string
		expected string
		hasError bool
	}{
		{
			name:     "Valid token with azp claim",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhenAiOiJ0ZXN0LWFwcCJ9.YqenXXXX", // token with azp: "test-app"
			expected: "test-app",
			hasError: false,
		},
		{
			name:     "Invalid token format",
			token:    "invalid-token",
			expected: "ast-app", // Should return default value
			hasError: false,
		},
		{
			name:     "Valid token without azp claim",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.XXXXX",
			expected: "ast-app", // Should return default value
			hasError: false,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractAZPFromToken(tt.token)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAPIKeyPayload(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "Valid token with azp claim",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhenAiOiJ0ZXN0LWFwcCJ9.YqenXXXX",
			expected: "grant_type=refresh_token&client_id=test-app&refresh_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhenAiOiJ0ZXN0LWFwcCJ9.YqenXXXX",
		},
		{
			name:     "Invalid token",
			token:    "invalid-token",
			expected: "grant_type=refresh_token&client_id=ast-app&refresh_token=invalid-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAPIKeyPayload(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetAgentNameAndOrigin(t *testing.T) {
	viper.Set(commonParams.AgentNameKey, "TestAgent")
	viper.Set(commonParams.OriginKey, "TestOrigin")
	commonParams.Version = "1.0.0"

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

	setAgentNameAndOrigin(req)

	userAgent := req.Header.Get("User-Agent")
	origin := req.Header.Get("Origin")

	expectedUserAgent := "TestAgent/1.0.0"
	if userAgent != expectedUserAgent {
		t.Errorf("User-Agent header mismatch: got %v, want %v", userAgent, expectedUserAgent)
	}

	expectedOrigin := "TestOrigin"
	if origin != expectedOrigin {
		t.Errorf("Origin header mismatch: got %v, want %v", origin, expectedOrigin)
	}
}
