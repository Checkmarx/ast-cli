//go:build !integration

package wrappers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryHTTPRequest_Success(t *testing.T) {
	fn := func() (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	}

	resp, err := retryHTTPRequest(fn, 4, 500*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRetryHTTPRequest_RetryOnBadGateway(t *testing.T) {
	attempts := 0
	fn := func() (*http.Response, error) {
		attempts++
		if attempts < 4 {
			return &http.Response{StatusCode: http.StatusBadGateway}, nil
		}
		return &http.Response{StatusCode: http.StatusOK}, nil
	}

	resp, err := retryHTTPRequest(fn, 4, 500*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 4, attempts)
}

func TestRetryHTTPRequest_FailAfterRetries(t *testing.T) {
	fn := func() (*http.Response, error) {
		return nil, errors.New("network error")
	}

	resp, err := retryHTTPRequest(fn, 4, 500*time.Millisecond)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRetryHTTPRequest_EndWithBadGateway(t *testing.T) {
	fn := func() (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway}, nil
	}

	resp, err := retryHTTPRequest(fn, 4, 500*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}
