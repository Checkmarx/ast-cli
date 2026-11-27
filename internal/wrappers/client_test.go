package wrappers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/credentialstore"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type fakeCredentialStore struct {
	data      map[string]string
	getErr    error
	setErr    error
	deleteErr error
}

func (f *fakeCredentialStore) Get(_ context.Context, credentialName string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	value, ok := f.data[credentialName]
	if !ok {
		return "", credentialstore.ErrNotFound
	}
	return value, nil
}

func (f *fakeCredentialStore) Set(_ context.Context, credentialName, value string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.data[credentialName] = value
	return nil
}

func (f *fakeCredentialStore) Delete(_ context.Context, credentialName string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.data, credentialName)
	return nil
}

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

	credentialsMutex.Lock()
	tokenStr := cachedAccessToken
	credentialsMutex.Unlock()
	assert.NotEmpty(t, tokenStr, "Token should not be empty")

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

// TestGetRealmURL_LoginOverrideSkipsStoredAPIKey guards the `cx auth login` fix:
// when ApikeyOverrideFlag is set, GetRealmURL must build the realm from the
// explicit --base-auth-uri/--tenant flags and must NOT decode the stored
// cx_apikey. A stale/malformed stored key previously surfaced here as a hard
// "failed to resolve IAM realm URL" error, making login impossible until the bad
// key was manually cleared.
// TestGetRealmURL_LoginOverrideSkipsStoredAPIKey guards the `cx auth login` fix:
// when ApikeyOverrideFlag is set, GetRealmURL must build the realm from the
// explicit --base-auth-uri/--tenant flags and must NOT decode the stored
// cx_apikey. A stale/malformed stored key previously surfaced here as a hard
// "failed to resolve IAM realm URL" error, making login impossible until the bad
// key was manually cleared.
func TestGetRealmURL_LoginOverrideSkipsStoredAPIKey(t *testing.T) {
	keys := []string{
		commonParams.ApikeyOverrideFlag,
		commonParams.BaseAuthURIKey,
		commonParams.TenantKey,
	}
	saved := make(map[string]interface{}, len(keys))
	for _, k := range keys {
		saved[k] = viper.Get(k)
	}
	t.Cleanup(func() {
		for _, k := range keys {
			viper.Set(k, saved[k])
		}
	})

	store := &fakeCredentialStore{data: map[string]string{}}
	t.Setenv("CX_CONFIG_FILE_PATH", filepath.Join(t.TempDir(), "checkmarxcli.yaml"))
	credentialstore.SetDefaultResolverForTest(credentialstore.NewResolver("checkmarxcli.yaml", credentialstore.PolicyAuto, store))
	t.Cleanup(credentialstore.ResetForTest)

	const malformedKey = "not-a-jwt" // single segment -> ExtractFromTokenClaims fails

	t.Run("override builds realm from flags despite a malformed stored key", func(t *testing.T) {
		viper.Set(commonParams.ApikeyOverrideFlag, true)
		store.data[credentialstore.CredentialAPIKey] = malformedKey
		viper.Set(commonParams.BaseAuthURIKey, "https://eu.iam.checkmarx.net")
		viper.Set(commonParams.TenantKey, "cx_seg")

		realmURL, err := GetRealmURL()

		assert.NoError(t, err)
		assert.Equal(t, "https://eu.iam.checkmarx.net/auth/realms/cx_seg", realmURL)
	})

	t.Run("without override a malformed stored key still errors (unchanged)", func(t *testing.T) {
		viper.Set(commonParams.ApikeyOverrideFlag, false)
		store.data[credentialstore.CredentialAPIKey] = malformedKey
		viper.Set(commonParams.BaseAuthURIKey, "https://eu.iam.checkmarx.net")
		viper.Set(commonParams.TenantKey, "cx_seg")

		_, err := GetRealmURL()

		assert.Error(t, err)
	})
}

func TestSetAgentNameAndOrigin(t *testing.T) {
	viper.Set(commonParams.AgentNameKey, "TestAgent")
	viper.Set(commonParams.OriginKey, "TestOrigin")
	commonParams.Version = "1.0.0"

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

	setAgentNameAndOrigin(req, false)

	userAgent := req.Header.Get("User-Agent")
	origin := req.Header.Get("origin")

	expectedUserAgent := "TestAgent/1.0.0"
	if userAgent != expectedUserAgent {
		t.Errorf("User-Agent header mismatch: got %v, want %v", userAgent, expectedUserAgent)
	}

	expectedOrigin := "TestOrigin"
	if origin != expectedOrigin {
		t.Errorf("Origin header mismatch: got %v, want %v", origin, expectedOrigin)
	}
}

// unsignedJWT builds a 3-segment JWT with the given claims. ExtractFromTokenClaims
// uses ParseUnverified, so the signature segment is irrelevant.
func unsignedJWT(claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + payload + ".sig"
}

func TestAgentHookTelemetryTimesOutWithoutRetrying(t *testing.T) {
	requests := make(chan string, 2)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests <- req.URL.Path
		<-release
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })
	for key, value := range map[string]any{
		commonParams.IgnoreProxyKey:        true,
		commonParams.ApikeyOverrideFlag:    false,
		commonParams.TokenExpirySecondsKey: 300,
	} {
		previous := viper.Get(key)
		t.Cleanup(func() { viper.Set(key, previous) })
		viper.Set(key, value)
	}
	credentialsMutex.Lock()
	previousToken, previousTime := cachedAccessToken, cachedAccessTime
	credentialsMutex.Unlock()
	t.Cleanup(func() {
		credentialsMutex.Lock()
		defer credentialsMutex.Unlock()
		cachedAccessToken, cachedAccessTime = previousToken, previousTime
	})
	writeCredentialsToCache(unsignedJWT(map[string]any{BaseURLKey: server.URL}))

	telemetry := NewHTTPTelemetryAIWrapper("telemetry").ForAgentHooks()
	err := telemetry.SendAIDataToLog(&DataForAITelemetry{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected telemetry request to time out, got %v", err)
	}
	if got := len(requests); got != 1 {
		t.Fatalf("got %d telemetry requests, want 1", got)
	}
	if path := <-requests; path != "/telemetry/log" {
		t.Fatalf("telemetry request used path %q", path)
	}
}

// TestExtractFromTokenClaims covers fix #10: the "aud" (and any) claim may be a
// string or an array of strings. The array form must not panic.
func TestExtractFromTokenClaims(t *testing.T) {
	const realm = "https://eu.iam.checkmarx.net/auth/realms/cx_seg"

	t.Run("string claim returned as-is", func(t *testing.T) {
		token := unsignedJWT(map[string]interface{}{"aud": realm})
		got, err := ExtractFromTokenClaims(token, "aud")
		assert.NoError(t, err)
		assert.Equal(t, realm, got)
	})

	t.Run("array claim returns first non-empty string (no panic)", func(t *testing.T) {
		token := unsignedJWT(map[string]interface{}{"aud": []interface{}{"", realm, "account"}})
		got, err := ExtractFromTokenClaims(token, "aud")
		assert.NoError(t, err)
		assert.Equal(t, realm, got)
	})

	t.Run("non-string, non-array claim errors instead of panicking", func(t *testing.T) {
		token := unsignedJWT(map[string]interface{}{"aud": 12345})
		_, err := ExtractFromTokenClaims(token, "aud")
		assert.Error(t, err)
	})

	t.Run("missing claim errors", func(t *testing.T) {
		token := unsignedJWT(map[string]interface{}{"iss": realm})
		_, err := ExtractFromTokenClaims(token, "aud")
		assert.Error(t, err)
	})
}

func TestRetryIAMHTTPRequest_Success(t *testing.T) {
	fn := func() (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockReadCloser{},
		}, nil
	}

	resp, err := retryHTTPForIAMRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRetryHTTPIAMRequest_RetryOnBadGateway(t *testing.T) {
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

	resp, err := retryHTTPForIAMRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, retryAttempts, attempts)
}

func TestRetryHTTPIAMRequest_RetryOnStatusNotImplemented(t *testing.T) {
	attempts := 0
	fn := func() (*http.Response, error) {
		attempts++
		if attempts < retryAttempts {
			return &http.Response{
				StatusCode: http.StatusNotImplemented,
				Body:       &mockReadCloser{},
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockReadCloser{},
		}, nil
	}

	resp, err := retryHTTPForIAMRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, retryAttempts, attempts)
}

func TestRetryHTTPIAMRequest_Fail(t *testing.T) {
	fn := func() (*http.Response, error) {
		return nil, errors.New("Resource Unavailable")
	}

	resp, err := retryHTTPForIAMRequest(fn, retryAttempts, retryDelay*time.Millisecond)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

// An unreachable OS keyring must surface as a keyring failure, never as a
// silent no-credentials path.
func TestConfigureClientCredentialsPropagatesKeyringUnavailable(t *testing.T) {
	t.Setenv("CX_CONFIG_FILE_PATH", filepath.Join(t.TempDir(), "checkmarxcli.yaml"))
	store := &fakeCredentialStore{
		data:   map[string]string{},
		getErr: fmt.Errorf("%w: dbus: failed to connect to socket", credentialstore.ErrKeyringUnavailable),
	}
	credentialstore.SetDefaultResolverForTest(credentialstore.NewResolver("checkmarxcli.yaml", credentialstore.PolicyAuto, store))
	t.Cleanup(credentialstore.ResetForTest)

	viper.Set(commonParams.PreferredCredentialTypeKey, "")

	_, err := configureClientCredentialsAndGetNewToken()
	assert.Error(t, err)
	assert.ErrorIs(t, err, credentialstore.ErrKeyringUnavailable)
}

func TestGetRealmURLPropagatesKeyringUnavailable(t *testing.T) {
	savedOverride := viper.Get(commonParams.ApikeyOverrideFlag)
	t.Cleanup(func() { viper.Set(commonParams.ApikeyOverrideFlag, savedOverride) })

	t.Setenv("CX_CONFIG_FILE_PATH", filepath.Join(t.TempDir(), "checkmarxcli.yaml"))
	store := &fakeCredentialStore{
		data:   map[string]string{},
		getErr: fmt.Errorf("%w: dbus: failed to connect to socket", credentialstore.ErrKeyringUnavailable),
	}
	credentialstore.SetDefaultResolverForTest(credentialstore.NewResolver("checkmarxcli.yaml", credentialstore.PolicyAuto, store))
	t.Cleanup(credentialstore.ResetForTest)

	viper.Set(commonParams.ApikeyOverrideFlag, false)

	_, err := GetRealmURL()
	assert.Error(t, err)
	assert.ErrorIs(t, err, credentialstore.ErrKeyringUnavailable)
}

func setProxyViperConfig(t *testing.T, authType, proxy string, ignoreProxy bool) {
	t.Helper()
	viper.Set(commonParams.ProxyTypeKey, authType)
	viper.Set(commonParams.ProxyKey, proxy)
	viper.Set(commonParams.IgnoreProxyKey, ignoreProxy)
	t.Cleanup(func() {
		viper.Set(commonParams.ProxyTypeKey, "")
		viper.Set(commonParams.ProxyKey, "")
		viper.Set(commonParams.IgnoreProxyKey, false)
	})
}

func TestValidateProxyAuthType_SupportedValues(t *testing.T) {
	supported := []string{
		"", "basic", "ntlm", "kerberos", "kerberos-native",
		"BASIC", "NTLM", "Ntlm", "Kerberos", "KERBEROS-NATIVE", " ntlm ",
	}
	for _, value := range supported {
		t.Run(fmt.Sprintf("%q", value), func(t *testing.T) {
			setProxyViperConfig(t, value, "", false)
			assert.NoError(t, ValidateProxyAuthType())
		})
	}
}

func TestValidateProxyAuthType_UnsupportedValues(t *testing.T) {
	unsupported := []string{"NotARealAuthType", "digest", "none", "ntlm2", "kerberos_native", "basic,ntlm"}
	for _, value := range unsupported {
		t.Run(value, func(t *testing.T) {
			setProxyViperConfig(t, value, "", false)
			err := ValidateProxyAuthType()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("invalid proxy authentication type %q", value))
			assert.Contains(t, err.Error(), "basic, ntlm, kerberos, kerberos-native")
			for _, location := range []string{"--proxy-auth-type", "CX_PROXY_AUTH_TYPE", "cx_proxy_auth_type"} {
				assert.Contains(t, err.Error(), location)
			}
		})
	}
}

func TestValidateProxyAuthType_ValidatesOnlyTheAuthType(t *testing.T) {
	const classicProxy = "http://classic.company.com:8080"

	setProxyViperConfig(t, "NTLM", classicProxy, false)
	assert.NoError(t, ValidateProxyAuthType())

	setProxyViperConfig(t, "", classicProxy, false)
	assert.NoError(t, ValidateProxyAuthType())

	setProxyViperConfig(t, "NotARealAuthType", classicProxy, true)
	assert.Error(t, ValidateProxyAuthType(), "an invalid type must be rejected even with --ignore-proxy")
}

func proxyTransport(t *testing.T) *http.Transport {
	t.Helper()
	transport, ok := GetClient(1).Transport.(*http.Transport)
	assert.True(t, ok, "expected an *http.Transport")
	return transport
}

func TestGetClient_ProxyAuthTypeSelectsClient(t *testing.T) {
	const classicProxy = "http://classic.company.com:8080"

	basicWithProxy := []string{"", "basic", "BASIC", "Basic", "NotARealAuthType"}
	for _, authType := range basicWithProxy {
		t.Run(fmt.Sprintf("basic client for %q", authType), func(t *testing.T) {
			setProxyViperConfig(t, authType, classicProxy, false)
			transport := proxyTransport(t)
			assert.Nil(t, transport.DialContext)
			assert.NotNil(t, transport.Proxy)
			req, err := http.NewRequest(http.MethodGet, "https://example.com", http.NoBody)
			assert.NoError(t, err)
			proxyURL, err := transport.Proxy(req)
			assert.NoError(t, err)
			assert.Equal(t, classicProxy, proxyURL.String(), "proxy address must be used as given")
		})
	}

	for _, authType := range []string{"ntlm", "NTLM", "Ntlm"} {
		t.Run(fmt.Sprintf("ntlm client for %q", authType), func(t *testing.T) {
			setProxyViperConfig(t, authType, classicProxy, false)
			transport := proxyTransport(t)
			assert.Nil(t, transport.Proxy)
			assert.NotNil(t, transport.DialContext)
		})
	}

	t.Run("ignore proxy bypasses the auth type", func(t *testing.T) {
		setProxyViperConfig(t, "NTLM", classicProxy, true)
		transport := proxyTransport(t)
		assert.Nil(t, transport.Proxy)
		assert.Nil(t, transport.DialContext)
	})
}

func TestGetClient_UnrecognizedProxyAuthTypeIsLoggedWhenVerbose(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	viper.Set(commonParams.DebugFlag, true)
	t.Cleanup(func() { viper.Set(commonParams.DebugFlag, false) })

	setProxyViperConfig(t, "NotARealAuthType", "http://classic.company.com:8080", false)
	GetClient(1)
	assert.Contains(t, logBuffer.String(), "falling back to basic")

	logBuffer.Reset()
	for _, authType := range []string{"", "basic", "BASIC"} {
		setProxyViperConfig(t, authType, "http://classic.company.com:8080", false)
		GetClient(1)
	}
	assert.NotContains(t, logBuffer.String(), "falling back to basic")
}

func TestExchangeLoginAPIKey_VerifiesSuppliedCredential(t *testing.T) {
	var apiKey string
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		if req.URL.Path != "/auth/realms/organization/"+BaseAuthURLSuffix {
			t.Errorf("unexpected request path: %s", req.URL.Path)
		}
		if err := req.ParseForm(); err != nil {
			t.Error(err)
		}
		if req.Form.Get("refresh_token") != apiKey {
			t.Error("request did not use the supplied API key")
		}
		if calls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"verified-access-token"}`))
	}))
	defer server.Close()
	apiKey = unsignedJWT(map[string]interface{}{"aud": server.URL + "/auth/realms/organization"})
	SetCachedAccessTokenForTest("different-cached-token")
	t.Cleanup(InvalidateAccessTokenCache)

	if token, err := ExchangeLoginAPIKey(apiKey); err == nil || token != "" {
		t.Fatal("IAM rejected the supplied key, so login must fail")
	}
	if token, err := ExchangeLoginAPIKey(apiKey); err != nil || token != "verified-access-token" {
		t.Fatalf("IAM accepted the supplied key, got error %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected two fresh token requests, got %d", calls)
	}
}
