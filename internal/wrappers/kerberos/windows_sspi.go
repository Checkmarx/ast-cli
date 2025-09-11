//go:build windows
// +build windows

package kerberos

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	// Import SSPI package - only compiled on Windows
	"github.com/alexbrainman/sspi/kerberos"
)

// WindowsSSPIDialContext creates a DialContext using Windows SSPI
func WindowsSSPIDialContext(dialer *net.Dialer, proxyURL *url.URL, proxySPN string, tlsConfig *tls.Config) func(ctx context.Context, network, addr string) (net.Conn, error) {
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
		return dialAndAuthenticateSSPI(addr, proxySPN, dialProxy)
	}
}

// dialAndAuthenticateSSPI handles the SSPI authentication flow
func dialAndAuthenticateSSPI(addr, proxySPN string, baseDial func() (net.Conn, error)) (net.Conn, error) {
	if proxySPN == "" {
		return nil, errors.New("Kerberos SPN is required for Windows native authentication")
	}

	conn, err := baseDial()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}

	// Get SPNEGO token using Windows SSPI
	token, err := getSSPIToken(proxySPN)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get SSPI token: %w", err)
	}

	// Send CONNECT request with Negotiate authentication
	if err := sendNegotiateConnect(conn, addr, token); err != nil {
		conn.Close()
		return nil, fmt.Errorf("proxy authentication failed: %w", err)
	}

	log.Printf("Successfully authenticated with proxy using Windows SSPI")
	return conn, nil
}

// sendNegotiateConnect sends HTTP CONNECT with Negotiate authentication
func sendNegotiateConnect(conn net.Conn, addr string, token []byte) error {
	header := make(http.Header)
	header.Set("Proxy-Connection", "Keep-Alive")

	if len(token) > 0 {
		tokenStr := base64.StdEncoding.EncodeToString(token)
		header.Set("Proxy-Authorization", "Negotiate "+tokenStr)
	}

	connect := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: header,
	}

	// Send request
	if err := connect.Write(conn); err != nil {
		return fmt.Errorf("failed to write CONNECT request: %w", err)
	}

	// Read response
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connect)
	if err != nil {
		return fmt.Errorf("failed to read CONNECT response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy returned status: %s", resp.Status)
	}

	return nil
}

// ValidateSSPISetup validates that Windows SSPI is available
func ValidateSSPISetup(proxySPN string) error {
	if proxySPN == "" {
		return errors.New("Kerberos SPN is required")
	}

	// Try to get a token to validate setup
	_, err := getSSPIToken(proxySPN)
	if err != nil {
		return fmt.Errorf("SSPI validation failed: %w", err)
	}

	return nil
}

// getSSPIToken gets a Kerberos SPNEGO token using Windows SSPI
func getSSPIToken(spn string) ([]byte, error) {
	// Acquire current user credentials using SSPI
	cred, err := kerberos.AcquireCurrentUserCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire Windows credentials: %w", err)
	}
	defer cred.Release()

	// Create security context for the SPN
	secCtx, _, token, err := kerberos.NewClientContext(cred, spn)
	if err != nil {
		return nil, fmt.Errorf("failed to create security context for SPN '%s': %w", spn, err)
	}
	defer secCtx.Release()

	if len(token) == 0 {
		return nil, errors.New("empty SPNEGO token received from SSPI")
	}

	log.Printf("Successfully generated SSPI token for SPN: %s", spn)
	return token, nil
}
