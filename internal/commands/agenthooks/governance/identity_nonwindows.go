//go:build !windows

package governance

// windowsMachineID is a no-op on non-Windows platforms.
// The caller falls through to /etc/machine-id or the YAML fallback.
func windowsMachineID() string { return "" }
