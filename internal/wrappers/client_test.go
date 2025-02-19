package wrappers

import (
	"errors"
	"fmt"
	"net/http"
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
