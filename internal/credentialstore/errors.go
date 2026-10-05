// Package credentialstore stores CLI credentials in the OS keyring with a plaintext
// config-file fallback.
//
// Note: unlike the HTTP wrappers, this package exposes package-level seams
// (Default/Resolve/Store/Clear) instead of constructor injection. Credential
// consumers include non-cobra entrypoints (agenthook dispatch, MCP bridge
// polling) where threading a wrapper through every constructor is impractical;
// the testing.go file provides the injection/reset seams this requires.
package credentialstore

import "errors"

var (
	// ErrNotFound is returned when a credential does not exist in any layer.
	ErrNotFound = errors.New("credential not found")
	// ErrKeyringUnavailable is returned when the OS keyring cannot be reached.
	ErrKeyringUnavailable = errors.New("credential keyring unavailable")
	// ErrAccessDenied is returned when the keyring refuses access to a credential.
	ErrAccessDenied = errors.New("access to credential denied")
	// ErrInvalidName is returned for an unknown credential name.
	ErrInvalidName = errors.New("unknown credential name")
	// ErrKeyringFailure wraps keyring errors that do not fit a known category.
	ErrKeyringFailure = errors.New("credential keyring failure")
)
