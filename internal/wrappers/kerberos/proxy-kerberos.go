package kerberos

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/jcmturner/gokrb5.v7/client"
	"gopkg.in/jcmturner/gokrb5.v7/config"
	"gopkg.in/jcmturner/gokrb5.v7/credentials"
	"gopkg.in/jcmturner/gokrb5.v7/spnego"
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

	// Load krb5.conf
	krb5cfg, err := config.Load(krb5ConfPath)
	if err != nil {
		log.Printf("Failed to load krb5.conf from %s: %s", krb5ConfPath, err)
		return conn, err
	}

	// Load credential cache
	ccPath := kerberosConfig.CcachePath
	if ccPath == "" {
		ccPath = getDefaultCCachePath()
	}

	cc, err := credentials.LoadCCache(ccPath)
	if err != nil {
		log.Printf("Failed to load ccache %s: %s", ccPath, err)
		return conn, err
	}

	// Create Kerberos client from cache
	krbClient, err := client.NewClientFromCCache(cc, krb5cfg)
	if err != nil {
		log.Printf("Failed to create Kerberos client: %s", err)
		return conn, err
	}

	// Step 1: Try plain request first (like curl does)
	header := make(http.Header)
	header.Set("Proxy-Connection", "Keep-Alive")
	connect := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: header,
	}

	err = connect.Write(conn)
	if err != nil {
		log.Printf("Could not write initial request to proxy: %s", err)
		return conn, err
	}

	// Read response
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connect)
	if err != nil {
		log.Printf("Could not read response from proxy: %s", err)
		return conn, err
	}

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Could not read response body from proxy: %s", err)
		return conn, err
	}
	_ = resp.Body.Close()

	// If proxy did NOT require auth, return connection
	if resp.StatusCode != http.StatusProxyAuthRequired {
		log.Printf("Proxy did not require authentication, connection established")
		return conn, nil
	}

	// Step 2: Proxy asked for auth. Check it includes Negotiate
	hasNegotiate := false
	for _, v := range resp.Header.Values("Proxy-Authenticate") {
		if strings.HasPrefix(strings.TrimSpace(v), "Negotiate") {
			hasNegotiate = true
			break
		}
	}
	if !hasNegotiate {
		log.Printf("Proxy requires auth but did not advertise Negotiate, got: '%s'",
			resp.Header.Get("Proxy-Authenticate"))
		return conn, fmt.Errorf("no Negotiate authentication method available")
	}

	// Step 3: Build SPNEGO token for the proxy SPN and attach as Proxy-Authorization
	header2 := make(http.Header) //nolint:gosec
	header2.Set("Proxy-Connection", "Keep-Alive")
	req2 := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: header2,
	}

	if err := spnego.SetSPNEGOHeader(krbClient, req2, kerberosConfig.ProxySPN); err != nil {
		log.Printf("Failed to set SPNEGO header: %s", err)
		return conn, err
	}

	// spnego.SetSPNEGOHeader sets Authorization: Negotiate <token>
	authVal := req2.Header.Get("Authorization")
	if authVal == "" {
		log.Printf("SPNEGO did not generate an Authorization header")
		return conn, fmt.Errorf("failed to generate SPNEGO token")
	}

	// Move Authorization -> Proxy-Authorization (proxy level)
	req2.Header.Set("Proxy-Authorization", authVal)
	req2.Header.Del("Authorization") // don't forward to origin

	// Step 4: Retry with Proxy-Authorization
	if err = req2.Write(conn); err != nil {
		log.Printf("Could not write authorization to proxy: %s", err)
		return conn, err
	}

	resp2, err := http.ReadResponse(br, req2)
	if err != nil {
		log.Printf("Could not read response from proxy: %s", err)
		return conn, err
	}

	if resp2.StatusCode != http.StatusOK {
		log.Printf("Expected %d as return status, got: %d", http.StatusOK, resp2.StatusCode)
		if resp2.StatusCode == http.StatusProxyAuthRequired {
			log.Printf("Proxy still returned 407 after sending Negotiate token. Check SPN and proxy keytab/KDC.")
			// Print Proxy-Authenticate for diagnostics
			for _, v := range resp2.Header.Values("Proxy-Authenticate") {
				log.Printf("Proxy-Authenticate: %s", v)
			}
		}
		_ = resp2.Body.Close()
		return conn, fmt.Errorf("proxy authentication failed: %s", resp2.Status)
	}

	// Successfully authorized with Kerberos
	_ = resp2.Body.Close()
	log.Printf("Successfully authenticated with proxy using Kerberos")
	return conn, nil
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
