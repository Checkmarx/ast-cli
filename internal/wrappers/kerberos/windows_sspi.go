//go:build windows
// +build windows

package kerberos

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/pkg/errors"

	// Import SSPI package - only compiled on Windows
	"github.com/alexbrainman/sspi/kerberos" //nolint
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
	conn, err := baseDial()
	if err != nil {
		return nil, errors.Errorf("Failed to connect to proxy: %v", err)
	}

	// Get SPNEGO token using Windows SSPI
	token, err := getSSPIToken(proxySPN)
	if err != nil {
		conn.Close()
		return nil, errors.Errorf("failed to get SSPI token: %v", err)
	}

	// Send CONNECT request with Negotiate authentication
	if err := sendNegotiateConnect(conn, addr, token); err != nil {
		conn.Close()
		return nil, errors.Errorf("proxy authentication failed: %v", err)
	}

	logger.PrintIfVerbose("Successfully authenticated with proxy using Windows SSPI")
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
		return errors.Errorf("failed to write CONNECT request: %w", err)
	}

	// Read response
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connect)
	if err != nil {
		return errors.Errorf("failed to read CONNECT response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("proxy returned status: %s", resp.Status)
	}

	defer resp.Body.Close()
	return nil
}

// ValidateSSPISetup validates that Windows SSPI is available
func ValidateSSPISetup(proxySPN string) error {

	// Try to get a token to validate setup
	_, err := getSSPIToken(proxySPN)
	if err != nil {
		return errors.Errorf("SSPI validation failed: %v", err)
	}

	return nil
}

// getSSPIToken gets a Kerberos SPNEGO token using Windows SSPI
func getSSPIToken(spn string) ([]byte, error) {
	// Acquire current user credentials using SSPI
	cred, err := kerberos.AcquireCurrentUserCredentials()
	if err != nil {
		return nil, errors.Errorf("failed to acquire Windows credentials: %w", err)
	}
	defer func() {
		_ = cred.Release()
	}()

	// Create security context for the SPN
	secCtx, _, token, err := kerberos.NewClientContext(cred, spn)
	if err != nil {
		return nil, errors.Errorf("failed to create security context for SPN '%s': %w", spn, err)
	}
	defer func() {
		_ = secCtx.Release()
	}()

	if len(token) == 0 {
		return nil, errors.New("empty SPNEGO token received from SSPI")
	}

	log.Printf("Successfully generated SSPI token for SPN: %s", spn)
	return token, nil
}
