//go:build darwin

package governance

import (
	"os/exec"
	"strings"
)

// darwinMachineID reads the stable platform UUID from the IOPlatformExpertDevice IOKit entry.
// Returns "" on any error so callers fall back to the persisted UUID fallback.
func darwinMachineID() string {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "IOPlatformUUID") {
			// Line looks like: "IOPlatformUUID" = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
			parts := strings.Split(line, `"`)
			if len(parts) >= 4 {
				return parts[len(parts)-2]
			}
		}
	}
	return ""
}
