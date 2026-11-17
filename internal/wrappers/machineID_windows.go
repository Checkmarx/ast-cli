//go:build windows
// +build windows

package wrappers

import (
	"github.com/checkmarx/ast-cli/internal/logger"
	"golang.org/x/sys/windows/registry"
)

func getWindowsMachineID() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		logger.PrintIfVerbose("Failed to open registry key: " + err.Error())
		return ""
	}
	defer k.Close()

	machineID, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		logger.PrintIfVerbose("Failed to read MachineGuid from registry: " + err.Error())
		return ""
	}
	return machineID
}
