//go:build !windows

package telemetry

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// listProcesses reads the process table via `ps`, available on both Linux and
// macOS. `etimes` (elapsed seconds) is used instead of a start timestamp because
// its format is identical across the two platforms, unlike `lstart`.
func listProcesses() ([]procInfo, error) {
	out, err := exec.Command("ps", "-eo", "pid=,etimes=,args=").Output()
	if err != nil {
		return nil, err
	}
	return parsePSOutput(string(out), time.Now()), nil
}

// parsePSOutput converts `ps -eo pid=,etimes=,args=` lines into procInfo rows.
func parsePSOutput(out string, now time.Time) []procInfo {
	var procs []procInfo
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		elapsed, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		// Rebuild the command line: everything after pid and etimes, with the
		// original spacing collapsed (matching is substring-based anyway).
		args := strings.Join(fields[2:], " ")
		procs = append(procs, procInfo{
			PID:       pid,
			CmdLine:   args,
			CreatedAt: now.Add(-time.Duration(elapsed) * time.Second),
		})
	}
	return procs
}

// hideConsole is a no-op outside Windows — POSIX has no console window to hide.
func hideConsole(_ *exec.Cmd) {}
