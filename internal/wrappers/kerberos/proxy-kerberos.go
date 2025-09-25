package kerberos

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jcmturner/gokrb5/v8/client"      //nolint
	"github.com/jcmturner/gokrb5/v8/config"      //nolint
	"github.com/jcmturner/gokrb5/v8/credentials" //nolint
	"github.com/jcmturner/gokrb5/v8/spnego"      //nolint
	"github.com/pkg/errors"
)

// DialContext is the DialContext function that should be wrapped with a
// Kerberos Authentication.
type DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

// KerberosConfig holds the configuration for Kerberos authentication
type KerberosConfig struct {
	ProxySPN     string // SPN configured in proxy keytab/KDC (must match)
	Krb5ConfPath string // path to krb5.conf file
	CcachePath   string // path to credential cache (optional, will use KRB5CCNAME or default)
}

// NewKerberosProxyDialContext creates a new DialContext that uses Kerberos authentication
// for proxy connections. Unlike NTLM, it describes the proxy location with a full URL,
// whose scheme can be HTTP or HTTPS.
func NewKerberosProxyDialContext(dialer *net.Dialer, proxyURL *url.URL,
	kerberosConfig KerberosConfig, tlsConfig *tls.Config) DialContext {
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialProxy := func() (net.Conn, error) {
			if proxyURL.Scheme == "https" {
				return tls.DialWithDialer(dialer, "tcp", proxyURL.Host, tlsConfig)
			}
			return dialer.DialContext(ctx, network, proxyURL.Host)
		}
		return dialAndNegotiate(addr, kerberosConfig, dialProxy)
	}
}

func dialAndNegotiate(addr string, kerberosConfig KerberosConfig, baseDial func() (net.Conn, error)) (net.Conn, error) {
	// Validate required SPN parameter early
	if kerberosConfig.ProxySPN == "" {
		log.Printf("Kerberos SPN is required but not provided")
		return nil, errors.New("Kerberos SPN is required. Use --proxy-kerberos-spn flag or CX_PROXY_KERBEROS_SPN env var")
	}

	conn, err := baseDial()
	if err != nil {
		log.Printf("Could not call dial context with proxy: %s", err)
		return conn, err
	}

	// Use default krb5.conf path if not specified
	krb5ConfPath := kerberosConfig.Krb5ConfPath
	if krb5ConfPath == "" {
		krb5ConfPath = GetDefaultKrb5ConfPath()
	}

	// Check if krb5.conf exists before trying to load it
	if _, err := os.Stat(krb5ConfPath); os.IsNotExist(err) {
		log.Printf("Kerberos configuration file not found at %s", krb5ConfPath)
		return conn, errors.New("Kerberos configuration file not found. Please ensure krb5.conf is properly configured")
	}

	// Load krb5.conf
	krb5cfg, err := config.Load(krb5ConfPath)
	if err != nil {
		log.Printf("Failed to load krb5.conf from %s: %s", krb5ConfPath, err)
		return conn, errors.New("failed to load Kerberos configuration. Please check the krb5.conf file")
	}

	// Load credential cache
	ccPath := kerberosConfig.CcachePath
	if ccPath == "" {
		ccPath = getDefaultCCachePath()
	}

	// Check if credential cache exists before trying to load it
	if ccPath != "" {
		if _, err := os.Stat(ccPath); os.IsNotExist(err) {
			log.Printf("Kerberos credential cache not found at %s", ccPath)
			return conn, errors.New("Kerberos credential cache not found. Please run 'kinit' to obtain Kerberos tickets first")
		}
	}

	cc, err := credentials.LoadCCache(ccPath)
	if err != nil {
		log.Printf("Failed to load Kerberos credential cache from %s: %s", ccPath, err)
		return conn, errors.New("failed to load Kerberos credential cache. Please run 'kinit' to obtain valid Kerberos tickets")
	}

	// Create Kerberos client from cache
	krbClient, err := client.NewFromCCache(cc, krb5cfg)
	if err != nil {
		log.Printf("Failed to create Kerberos client: %s", err)
		return conn, errors.New("failed to create Kerberos client. Please check your Kerberos tickets with 'klist'")
	}

	// Kerberos Step 1: Send CONNECT with SPNEGO token directly (like NTLM does)
	header := make(http.Header) //nolint:gosec
	header.Set("Proxy-Connection", "Keep-Alive")
	connect := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: header,
	}

	// Build SPNEGO token for the proxy SPN
	if err := spnego.SetSPNEGOHeader(krbClient, connect, kerberosConfig.ProxySPN); err != nil {
		log.Printf("Failed to generate SPNEGO token for SPN '%s': %s", kerberosConfig.ProxySPN, err)
		return conn, errors.New("failed to generate SPNEGO token. Please check if the SPN is correct")
	}

	// spnego.SetSPNEGOHeader sets Authorization: Negotiate <token>
	authVal := connect.Header.Get("Authorization")
	if authVal == "" {
		log.Printf("SPNEGO did not generate an Authorization header")
		return conn, errors.New("failed to generate SPNEGO token. Please check Kerberos configuration")
	}

	// Move Authorization -> Proxy-Authorization (proxy level)
	connect.Header.Set("Proxy-Authorization", authVal)
	connect.Header.Del("Authorization") // don't forward to origin

	log.Printf("Sending CONNECT with Kerberos SPNEGO token")
	err = connect.Write(conn)
	if err != nil {
		log.Printf("Could not write Kerberos auth request to proxy: %s", err)
		return conn, err
	}

	// Kerberos Step 2: Read response
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connect)
	if err != nil {
		log.Printf("Could not read response from proxy: %s", err)
		return conn, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Expected %d as return status, got: %d", http.StatusOK, resp.StatusCode)
		if resp.StatusCode == http.StatusProxyAuthRequired {
			log.Printf("Proxy returned 407 after sending Negotiate token. Check SPN and proxy keytab/KDC.")
			// Print Proxy-Authenticate for diagnostics
			for _, v := range resp.Header.Values("Proxy-Authenticate") {
				log.Printf("Proxy-Authenticate: %s", v)
			}
			_ = resp.Body.Close()
			return conn, errors.New("proxy authentication failed. Check SPN and proxy keytab/KDC configuration")
		}
		_ = resp.Body.Close()
		return conn, errors.New(http.StatusText(resp.StatusCode))
	}

	// Successfully authorized with Kerberos
	_ = resp.Body.Close()
	log.Printf("Successfully authenticated with proxy using Kerberos")
	return conn, nil
}

// ValidateKerberosSetup validates Kerberos configuration early to fail fast
// This function performs all the same checks as dialAndNegotiate but without actually
// making network connections, allowing early detection of configuration problems
func ValidateKerberosSetup(krb5ConfPath, ccachePath, proxySPN string) error {
	// Validate SPN
	if proxySPN == "" {
		return errors.New("Kerberos SPN is required. Use --proxy-kerberos-spn flag or CX_PROXY_KERBEROS_SPN env var")
	}

	// Use default krb5.conf path if not specified
	if krb5ConfPath == "" {
		krb5ConfPath = GetDefaultKrb5ConfPath()
	}

	// Check if krb5.conf exists
	if _, err := os.Stat(krb5ConfPath); os.IsNotExist(err) {
		return errors.New("Kerberos configuration file not found. Please ensure krb5.conf is properly configured")
	}

	// Load krb5.conf to validate it's readable
	_, err := config.Load(krb5ConfPath)
	if err != nil {
		return errors.New("failed to load Kerberos configuration. Please check the krb5.conf file")
	}

	// Get default credential cache path if not specified
	if ccachePath == "" {
		ccachePath = getDefaultCCachePath()
	}

	// Check if credential cache exists
	if ccachePath != "" {
		if _, err := os.Stat(ccachePath); os.IsNotExist(err) {
			return errors.New("Kerberos credential cache not found. Please run 'kinit' to obtain Kerberos tickets first")
		}
	}

	// Try to load credential cache to validate it's usable
	cc, err := credentials.LoadCCache(ccachePath)
	if err != nil {
		return errors.New("failed to load Kerberos credential cache. Please run 'kinit' to obtain valid Kerberos tickets")
	}

	// Try to create Kerberos client to validate tickets are valid
	krb5cfg, err := config.Load(krb5ConfPath)
	if err != nil {
		return errors.New("failed to reload Kerberos configuration")
	}

	_, err = client.NewFromCCache(cc, krb5cfg)
	if err != nil {
		return errors.New("failed to create Kerberos client. Please check your Kerberos tickets with 'klist'")
	}

	return nil
}

// GetDefaultKrb5ConfPath returns the default krb5.conf path for the current platform
func GetDefaultKrb5ConfPath() string {
	switch runtime.GOOS {
	case "windows":
		// Windows typically uses krb5.ini
		if windir := os.Getenv("WINDIR"); windir != "" {
			return filepath.Join(windir, "krb5.ini")
		}
		// Fallback locations
		locations := []string{
			"C:\\ProgramData\\MIT\\Kerberos5\\krb5.ini",
			"C:\\Windows\\krb5.ini",
		}
		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				return loc
			}
		}
		return "C:\\Windows\\krb5.ini" // Default fallback
	default:
		// Linux, macOS, and other Unix-like systems
		return "/etc/krb5.conf"
	}
}

// getDefaultCCachePath returns the default credential cache path for the current platform
func getDefaultCCachePath() string {
	// Check KRB5CCNAME environment variable first
	if ccname := os.Getenv("KRB5CCNAME"); ccname != "" {
		return ccname
	}

	switch runtime.GOOS {
	case "windows":
		// On Windows, use the default credential cache managed by the system
		// The gokrb5 library should handle this automatically with empty string
		return ""
	default:
		// Linux, macOS, and other Unix-like systems
		return fmt.Sprintf("/tmp/krb5cc_%d", os.Getuid())
	}
}
