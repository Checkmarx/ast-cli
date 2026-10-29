package commands

import (
	"archive/zip"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/filtering"
	"github.com/checkmarx/ast-cli/internal/logger"
)

const (
	globMetaChars  = `*?[\`
	zipCopyBufSize = 32 * 1024
)

// fileMatcher is a precompiled, case-insensitive equivalent of filterMatched.
// Patterns are lower-cased and classified once, so matching a file name costs a few map
// lookups instead of one path.Match per pattern, whatever the size of the filter list.
type fileMatcher struct {
	hasInclude bool
	exact      map[string]struct{} // "makefile", "go.mod"
	suffix     map[string]struct{} // "*.java" -> ".java", "*.auto.tfvars" -> ".auto.tfvars"
	globs      []string            // any other pattern, e.g. "*dockerfile*", "*.php5*6"
	exclusions []string            // "!node_modules" -> "node_modules"
}

func newFileMatcher(filters []string) *fileMatcher {
	m := &fileMatcher{exact: map[string]struct{}{}, suffix: map[string]struct{}{}}
	for _, filter := range filters {
		if filter == "" {
			continue
		}
		filter = strings.ToLower(filter)
		if filter[0] == '!' {
			m.exclusions = append(m.exclusions, filter[1:])
			continue
		}
		m.hasInclude = true
		switch {
		case !strings.ContainsAny(filter, globMetaChars):
			m.exact[filter] = struct{}{}
		case strings.HasPrefix(filter, "*.") && !strings.ContainsAny(filter[1:], globMetaChars):
			m.suffix[filter[1:]] = struct{}{}
		default:
			m.globs = append(m.globs, filter)
		}
	}
	return m
}

// Matched reports whether fileName passes the filters: it must not match any exclusion and,
// when inclusions exist, must match at least one of them.
func (m *fileMatcher) Matched(fileName string) bool {
	name := strings.ToLower(fileName)
	for _, exclusion := range m.exclusions {
		if excluded, _ := path.Match(exclusion, name); excluded {
			return false
		}
	}
	if !m.hasInclude {
		return true
	}
	if _, ok := m.exact[name]; ok {
		return true
	}
	// Try every suffix that starts at a dot, so "x.auto.tfvars" checks ".auto.tfvars" and ".tfvars".
	for i := strings.IndexByte(name, '.'); i >= 0; {
		if _, ok := m.suffix[name[i:]]; ok {
			return true
		}
		next := strings.IndexByte(name[i+1:], '.')
		if next < 0 {
			break
		}
		i += next + 1
	}
	for _, glob := range m.globs {
		if matched, _ := path.Match(glob, name); matched {
			return true
		}
	}
	return false
}

// zipWalker carries what the source walk needs to decide what goes into the zip.
// It is built once per scan in compressFolder.
type zipWalker struct {
	zipWriter      *zip.Writer
	excludeFilters []string // raw exclude list, still used by isDirFiltered for directory names
	include        *fileMatcher
	exclude        *fileMatcher
	antMatcher     filtering.Matcher
	excludeGitDir  bool
	verbose        bool
	buf            []byte
}

func newZipWalker(zipWriter *zip.Writer, excludeFilters, includeFilters []string, antMatcher filtering.Matcher, excludeGitDir, verbose bool) *zipWalker {
	return &zipWalker{
		zipWriter:      zipWriter,
		excludeFilters: excludeFilters,
		include:        newFileMatcher(includeFilters),
		exclude:        newFileMatcher(excludeFilters),
		antMatcher:     antMatcher,
		excludeGitDir:  excludeGitDir,
		verbose:        verbose,
		buf:            make([]byte, zipCopyBufSize),
	}
}

// fileIncluded mirrors the original filterMatched(include) && filterMatched(exclude) check.
func (w *zipWalker) fileIncluded(fileName, relPath string) bool {
	return w.include.Matched(fileName) && w.exclude.Matched(fileName) && !w.antMatcher.Excluded(relPath, false)
}

func (w *zipWalker) printIfVerbose(msg string) {
	if w.verbose {
		logger.PrintIfVerbose(msg)
	}
}

// copyFileToZip streams src into a new zip entry without loading the whole file into memory.
// The entry is created only after src opens, so an unreadable file leaves no empty entry.
func copyFileToZip(zipWriter *zip.Writer, entryName, src string, buf []byte) error {
	f, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer f.Close()
	entry, err := zipWriter.Create(entryName)
	if err != nil {
		return err
	}
	// Hide *os.File's WriterTo so CopyBuffer uses buf instead of allocating a buffer per file.
	_, err = io.CopyBuffer(entry, struct{ io.Reader }{f}, buf)
	return err
}
