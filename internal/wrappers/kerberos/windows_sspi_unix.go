//go:build !windows
// +build !windows

package kerberos

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"

	"github.com/pkg/errors"
)

// WindowsSSPIDialContext creates a DialContext using Windows SSPI (Unix stub)
func WindowsSSPIDialContext(dialer *net.Dialer, proxyURL *url.URL, proxySPN string, tlsConfig *tls.Config) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, errors.New("Windows SSPI is not available on this platform")
	}
}

// ValidateSSPISetup validates that Windows SSPI is available (Unix stub)
func ValidateSSPISetup(proxySPN string) error {
	return errors.New("Windows SSPI is not available on this platform")
}
