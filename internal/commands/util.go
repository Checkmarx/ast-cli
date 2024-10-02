package commands

import (
	"path/filepath"
	"strings"
)

func isFilePath(s string) bool {
	return strings.Contains(s, string(filepath.Separator)) || strings.Contains(s, "/") || strings.Contains(s, "\\")
}
