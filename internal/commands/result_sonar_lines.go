package commands

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// maxSonarLineBytes bounds how long a single source line may be before the file
// is treated as unreadable. Lines beyond this do not warrant column precision.
const maxSonarLineBytes = 1024 * 1024

// byteOrderMarkRune is U+FEFF. A leading BOM must not be counted as a
// character, or every offset on the first line would shift by one.
const byteOrderMarkRune rune = 0xFEFF

// parentDir is the path element that walks above a directory.
const parentDir = ".."

// lineStatus reports how much of a location the source file could confirm.
type lineStatus int

const (
	// lineStatusFileUnknown means the file could not be read, so neither the
	// line nor the columns can be verified.
	lineStatusFileUnknown lineStatus = iota
	// lineStatusLineMissing means the file was read and does not contain the
	// reported line. The location is definitively invalid.
	lineStatusLineMissing
	// lineStatusOK means the reported line exists and its length is known.
	lineStatusOK
)

// sonarLineIndex caches the character length of every line of the source files
// referenced by a report, so that each file is read at most once per export.
//
// SonarQube validates both the line and the column offsets of every imported
// issue against the file on disk, and aborts the entire analysis on the first
// violation. The CxOne engine reports coordinates against its own view of the
// source, which is not guaranteed to match the file on disk - minified assets
// are a common example, where the engine reports lines far beyond the end of
// the raw file. Locations therefore have to be verified before being written.
type sonarLineIndex struct {
	// baseDir confines every file read to the working directory, matching how
	// SonarQube resolves filePath against sonar.projectBaseDir. An empty
	// baseDir disables verification entirely.
	baseDir string
	// files maps a report file path to the character length of each of its
	// lines. A nil value records that the file could not be read, so an
	// unreadable file is not reopened for every node.
	files map[string][]uint
}

func newSonarLineIndex() *sonarLineIndex {
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = ""
	}
	return &sonarLineIndex{baseDir: baseDir, files: make(map[string][]uint)}
}

// resolveLine reports whether the given 1-based line of fileName exists and,
// when it does, how many characters it holds.
func (index *sonarLineIndex) resolveLine(fileName string, line uint) (length uint, status lineStatus) {
	if index == nil || fileName == "" {
		return 0, lineStatusFileUnknown
	}

	lengths, cached := index.files[fileName]
	if !cached {
		lengths = index.readLineLengths(fileName)
		index.files[fileName] = lengths
	}
	if lengths == nil {
		return 0, lineStatusFileUnknown
	}

	if line == 0 || line > uint(len(lengths)) {
		return 0, lineStatusLineMissing
	}
	return lengths[line-1], lineStatusOK
}

// resolveSourcePath cleans a report file path and confines it to baseDir,
// rejecting anything that escapes. Report paths are repository relative and
// arrive in a scan-result payload, so they are treated as untrusted input.
// Symlinks are resolved so that containment is enforced on real paths rather
// than lexically.
func (index *sonarLineIndex) resolveSourcePath(fileName string) (path string, ok bool) {
	if index.baseDir == "" {
		return "", false
	}

	// Reject anything that is not a plain relative path before touching disk.
	relative := filepath.FromSlash(strings.TrimLeft(fileName, "/\\"))
	if relative == "" || filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" {
		return "", false
	}

	// filepath.Join applies filepath.Clean, collapsing any ".." elements.
	cleaned := filepath.Join(index.baseDir, relative)

	// Resolve both sides: a path that cannot be resolved does not exist and is
	// therefore not readable either way.
	realBase, err := filepath.EvalSymlinks(index.baseDir)
	if err != nil {
		return "", false
	}
	realPath, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return "", false
	}

	inside, err := filepath.Rel(realBase, realPath)
	if err != nil || inside == parentDir || strings.HasPrefix(inside, parentDir+string(os.PathSeparator)) {
		return "", false
	}
	return realPath, true
}

// readLineLengths returns the character length of each line of a file, or nil
// when the path is not permitted or the file cannot be read in full.
func (index *sonarLineIndex) readLineLengths(fileName string) []uint {
	// resolveSourcePath validates and cleans the path and confines it to
	// baseDir, so the value opened below is not attacker controlled.
	path, ok := index.resolveSourcePath(fileName)
	if !ok {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxSonarLineBytes)

	lengths := []uint{}
	for scanner.Scan() {
		text := scanner.Text()
		if len(lengths) == 0 {
			text = strings.TrimPrefix(text, string(byteOrderMarkRune))
		}
		// bufio.ScanLines already drops a trailing CR; this guards against a
		// lone CR surviving into the count.
		text = strings.TrimSuffix(text, "\r")
		lengths = append(lengths, uint(utf8.RuneCountInString(text)))
	}
	if scanner.Err() != nil {
		return nil
	}
	return lengths
}

// clampSonarColumns constrains a start offset and length to a line of
// lineLength characters. emit is false when no valid column range exists for
// the line, in which case the caller should emit a line-level location.
func clampSonarColumns(startColumn, length, lineLength uint) (start, end uint, emit bool) {
	if lineLength == 0 {
		return 0, 0, false
	}

	start = startColumn
	if start > lineLength {
		start = lineLength
	}

	end = start + length
	if end > lineLength {
		end = lineLength
	}

	// A zero-length node, or a start offset sitting at end of line, leaves no
	// forward span. Highlight the whole line rather than emitting a zero-width
	// or inverted pointer, which SonarQube also rejects.
	if end <= start {
		return 0, lineLength, true
	}
	return start, end, true
}
