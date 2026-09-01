//go:build !darwin

package governance

// darwinMachineID is a no-op on non-macOS platforms.
func darwinMachineID() string { return "" }
