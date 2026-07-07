//go:build !integration

package commands

import (
	"errors"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

// unlicensedJWT authenticates fine but reports no AI-feature license for any
// engine (IsAllowedEngine → false, nil). The embedded mock supplies the rest of
// the JWTWrapper interface.
type unlicensedJWT struct{ *mock.JWTMockWrapper }

func (unlicensedJWT) IsAllowedEngine(string) (bool, error) { return false, nil }

// unauthenticatedJWT cannot authenticate the credential at all
// (IsAllowedEngine → error).
type unauthenticatedJWT struct{ *mock.JWTMockWrapper }

func (unauthenticatedJWT) IsAllowedEngine(string) (bool, error) {
	return false, errors.New("failed to authenticate")
}

func TestScannerAuth_States(t *testing.T) {
	cases := []struct {
		name     string
		jwt      wrappers.JWTWrapper
		want     scannerAuthState
		licensed bool
	}{
		{"ready", &mock.JWTMockWrapper{}, scannerReady, true},
		{"unlicensed", unlicensedJWT{&mock.JWTMockWrapper{}}, scannerUnlicensed, false},
		{"unauthenticated", unauthenticatedJWT{&mock.JWTMockWrapper{}}, scannerUnauthenticated, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scannerAuth(tc.jwt); got != tc.want {
				t.Fatalf("scannerAuth() = %v, want %v", got, tc.want)
			}
			// isLicensed must stay consistent with the ready state (no regression
			// for existing dispatch/MCP callers).
			if got := isLicensed(tc.jwt); got != tc.licensed {
				t.Errorf("isLicensed() = %v, want %v", got, tc.licensed)
			}
		})
	}
}

func TestCheckAuthExitCode(t *testing.T) {
	cases := map[scannerAuthState]int{
		scannerReady:           checkAuthExitReady,
		scannerUnlicensed:      checkAuthExitUnlicensed,
		scannerUnauthenticated: checkAuthExitUnauthenticated,
	}
	for state, want := range cases {
		if got := checkAuthExitCode(state); got != want {
			t.Errorf("checkAuthExitCode(%v) = %d, want %d", state, got, want)
		}
	}
}

func TestBuildCheckAuthResult(t *testing.T) {
	ready := buildCheckAuthResult(scannerReady)
	if !ready.ScannerReady || !ready.Authenticated || !ready.Licensed || ready.State != "ready" {
		t.Errorf("ready result wrong: %+v", ready)
	}
	unlic := buildCheckAuthResult(scannerUnlicensed)
	if unlic.ScannerReady || !unlic.Authenticated || unlic.Licensed || unlic.State != "unlicensed" {
		t.Errorf("unlicensed result wrong: %+v", unlic)
	}
	unauth := buildCheckAuthResult(scannerUnauthenticated)
	if unauth.ScannerReady || unauth.Authenticated || unauth.Licensed || unauth.State != "unauthenticated" {
		t.Errorf("unauthenticated result wrong: %+v", unauth)
	}
}
