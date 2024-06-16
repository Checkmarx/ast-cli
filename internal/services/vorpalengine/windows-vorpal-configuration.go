//go:build windows

package vorpalengine

import (
	"os/exec"
	"syscall"
)

func configureIndependentProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
