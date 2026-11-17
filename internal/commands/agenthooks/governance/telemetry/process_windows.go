//go:build windows

package telemetry

import (
	"encoding/csv"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// listProcesses reads the Windows process table via CIM.
//
// `tasklist` is cheaper but omits command lines, which is the only field that
// can tie a process back to the shell command an agent submitted — so CIM it is.
// Output is CSV rather than JSON because ConvertTo-Csv is stable across the
// Windows PowerShell / PowerShell 7 split and never wraps single rows in an
// object instead of an array.
func listProcesses() ([]procInfo, error) {
	const script = `Get-CimInstance Win32_Process | ` +
		`Select-Object ProcessId,CommandLine,` +
		`@{n='Created';e={if($_.CreationDate){$_.CreationDate.ToUniversalTime().ToString('o')}else{''}}} | ` +
		`ConvertTo-Csv -NoTypeInformation`

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseProcessCSV(string(out)), nil
}

// parseProcessCSV turns the CIM CSV dump into procInfo rows, skipping any row
// the reader cannot make sense of.
func parseProcessCSV(out string) []procInfo {
	r := csv.NewReader(strings.NewReader(out))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil || len(records) < 2 {
		return nil
	}
	procs := make([]procInfo, 0, len(records)-1)
	for _, rec := range records[1:] {
		if len(rec) < 3 {
			continue
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
		created, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(rec[2]))
		procs = append(procs, procInfo{
			PID:       pid,
			CmdLine:   rec[1],
			CreatedAt: created,
		})
	}
	return procs
}

// hideConsole keeps a detached watcher from flashing a console window.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
