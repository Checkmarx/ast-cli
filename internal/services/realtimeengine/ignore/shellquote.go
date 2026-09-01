package ignore

import (
	"runtime"
	"strings"
)

// goosWindows is runtime.GOOS's value on Windows, factored out because the shell-quoting
// check below (and its test) compare against it repeatedly.
const goosWindows = "windows"

// QuoteDataFlag formats finding JSON for a shell --data argument.
// On Windows, PowerShell strips embedded double quotes when invoking native
// executables, yielding invalid JSON like {FileName:...}; inner quotes must be
// backslash-escaped inside a single-quoted argument.
func QuoteDataFlag(data []byte) string {
	s := string(data)
	if runtime.GOOS == goosWindows {
		return "'" + strings.ReplaceAll(s, `"`, `\"`) + "'"
	}
	return "'" + s + "'"
}
