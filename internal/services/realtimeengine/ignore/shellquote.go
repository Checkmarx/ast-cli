package ignore

import (
	"runtime"
	"strings"
)

// QuoteDataFlag formats finding JSON for a shell --data argument.
// On Windows, PowerShell strips embedded double quotes when invoking native
// executables, yielding invalid JSON like {FileName:...}; inner quotes must be
// backslash-escaped inside a single-quoted argument.
func QuoteDataFlag(data []byte) string {
	s := string(data)
	if runtime.GOOS == "windows" {
		return "'" + strings.ReplaceAll(s, `"`, `\"`) + "'"
	}
	return "'" + s + "'"
}
