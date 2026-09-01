//go:build windows

package governance

import "golang.org/x/sys/windows/registry"

// windowsMachineID reads the stable Windows machine GUID from the registry.
// Returns "" on any error so callers fall back to the next resolution strategy.
func windowsMachineID() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Cryptography`,
		registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	guid, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return ""
	}
	return guid
}
