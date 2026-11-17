//go:build !windows
// +build !windows

package wrappers

func getWindowsMachineID() string {
	return ""
}
