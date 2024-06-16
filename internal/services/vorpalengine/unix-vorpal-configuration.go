//go:build linux || darwin

package vorpalengine

import (
	"os/exec"
	"syscall"
)

func configureIndependentProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}
