package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/stretchr/testify/assert"
)

// writeSourceFile creates a file with the given content inside dir and returns
// its path relative to dir, which is the shape report file paths take.
func writeSourceFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return filepath.ToSlash(name)
}

// assertValidSonarRange asserts the invariants SonarScanner enforces in
// DefaultInputFile: the line must exist in the file, and every offset must fall
// inside that line with the range moving forward. Violating any of them aborts
// the whole analysis, so totalLines is checked as well as lineLength.
func assertValidSonarRange(t *testing.T, tr *wrappers.SonarTextRange, totalLines, lineLength uint) {
	t.Helper()
	if tr == nil {
		return // omitted entirely, so the issue is file level and always valid
	}
	assert.NotZero(t, tr.StartLine, "startLine must be set when textRange is present")
	assert.LessOrEqual(t, tr.StartLine, totalLines, "startLine must exist in the file")

	if tr.StartColumn == 0 && tr.EndColumn == 0 {
		return // line level range, always accepted
	}
	assert.LessOrEqual(t, tr.StartColumn, lineLength, "startColumn must not exceed line length")
	assert.LessOrEqual(t, tr.EndColumn, lineLength, "endColumn must not exceed line length")
	assert.Less(t, tr.StartColumn, tr.EndColumn, "range must move forward")
}

func TestClampSonarColumns(t *testing.T) {
	tests := []struct {
		name        string
		startColumn uint
		length      uint
		lineLength  uint
		wantStart   uint
		wantEnd     uint
		wantEmit    bool
	}{
		{
			// The reported customer case: "@Component({" is 12 characters and
			// the engine reported Column=12, Length=6, producing endColumn 17.
			name:        "overflowing end column is clamped to line length",
			startColumn: 11, length: 6, lineLength: 12,
			wantStart: 11, wantEnd: 12, wantEmit: true,
		},
		{
			name:        "valid range is preserved unchanged",
			startColumn: 4, length: 5, lineLength: 40,
			wantStart: 4, wantEnd: 9, wantEmit: true,
		},
		{
			name:        "range ending exactly at line length is valid",
			startColumn: 0, length: 12, lineLength: 12,
			wantStart: 0, wantEnd: 12, wantEmit: true,
		},
		{
			name:        "zero length node falls back to the whole line",
			startColumn: 4, length: 0, lineLength: 40,
			wantStart: 0, wantEnd: 40, wantEmit: true,
		},
		{
			name:        "start beyond end of line falls back to the whole line",
			startColumn: 50, length: 3, lineLength: 12,
			wantStart: 0, wantEnd: 12, wantEmit: true,
		},
		{
			name:        "empty line yields no column range",
			startColumn: 0, length: 3, lineLength: 0,
			wantStart: 0, wantEnd: 0, wantEmit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, emit := clampSonarColumns(tt.startColumn, tt.length, tt.lineLength)
			assert.Equal(t, tt.wantEmit, emit)
			assert.Equal(t, tt.wantStart, start)
			assert.Equal(t, tt.wantEnd, end)
			if emit {
				assert.LessOrEqual(t, end, tt.lineLength)
				assert.Less(t, start, end)
			}
		})
	}
}

func TestSonarLineIndexLineLength(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	lfFile := writeSourceFile(t, dir, "lf.ts", "one\nthree3\n")
	crlfFile := writeSourceFile(t, dir, "crlf.ts", "one\r\nthree3\r\n")
	bomFile := writeSourceFile(t, dir, "bom.ts", string(byteOrderMarkRune)+"abc\n")
	utf8File := writeSourceFile(t, dir, "utf8.ts", "héllo→\n")
	emptyLineFile := writeSourceFile(t, dir, "nested/dir/empty.ts", "\nabc\n")

	index := newSonarLineIndex()

	t.Run("counts characters on an LF file", func(t *testing.T) {
		length, status := index.resolveLine(lfFile, 2)
		assert.Equal(t, lineStatusOK, status)
		assert.Equal(t, uint(6), length)
	})

	t.Run("CRLF terminator is not counted", func(t *testing.T) {
		length, status := index.resolveLine(crlfFile, 2)
		assert.Equal(t, lineStatusOK, status)
		assert.Equal(t, uint(6), length)
	})

	t.Run("leading BOM is not counted", func(t *testing.T) {
		length, status := index.resolveLine(bomFile, 1)
		assert.Equal(t, lineStatusOK, status)
		assert.Equal(t, uint(3), length)
	})

	t.Run("multi byte characters count as one character each", func(t *testing.T) {
		length, status := index.resolveLine(utf8File, 1)
		assert.Equal(t, lineStatusOK, status)
		// "héllo→" is 6 characters but 9 bytes.
		assert.Equal(t, uint(6), length)
	})

	t.Run("empty line reports zero length", func(t *testing.T) {
		length, status := index.resolveLine(emptyLineFile, 1)
		assert.Equal(t, lineStatusOK, status)
		assert.Equal(t, uint(0), length)
	})

	t.Run("leading slash in the report path is tolerated", func(t *testing.T) {
		length, status := index.resolveLine("/"+lfFile, 1)
		assert.Equal(t, lineStatusOK, status)
		assert.Equal(t, uint(3), length)
	})

	t.Run("line past end of a readable file is reported missing", func(t *testing.T) {
		_, status := index.resolveLine(lfFile, 99)
		assert.Equal(t, lineStatusLineMissing, status)
	})

	t.Run("line zero of a readable file is reported missing", func(t *testing.T) {
		_, status := index.resolveLine(lfFile, 0)
		assert.Equal(t, lineStatusLineMissing, status)
	})

	t.Run("missing file is reported unknown, not missing line", func(t *testing.T) {
		_, status := index.resolveLine("does/not/exist.ts", 1)
		assert.Equal(t, lineStatusFileUnknown, status)
	})

	t.Run("empty file name is reported unknown", func(t *testing.T) {
		_, status := index.resolveLine("", 1)
		assert.Equal(t, lineStatusFileUnknown, status)
	})
}

func TestSonarLineIndexRejectsPathsOutsideBaseDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	index := newSonarLineIndex()

	for _, fileName := range []string{
		"../escape.ts",
		"../../../../etc/passwd",
		"nested/../../escape.ts",
	} {
		t.Run("rejects "+fileName, func(t *testing.T) {
			_, ok := index.resolveSourcePath(fileName)
			assert.False(t, ok, "path outside the working directory must be rejected")
		})
	}

	t.Run("rejects an absolute path", func(t *testing.T) {
		absolute := filepath.Join(dir, "abs.ts")
		assert.NoError(t, os.WriteFile(absolute, []byte("abc\n"), 0o600))
		// Absolute inputs are refused even when they resolve inside baseDir.
		_, ok := index.resolveSourcePath(absolute)
		assert.False(t, ok)
	})

	t.Run("accepts a plain relative path inside the base directory", func(t *testing.T) {
		name := writeSourceFile(t, dir, "inside.ts", "abc\n")
		resolved, ok := index.resolveSourcePath(name)
		assert.True(t, ok)
		assert.True(t, strings.HasSuffix(resolved, "inside.ts"))
	})
}

// TestParseSonarTextRangeCustomerRegression reproduces the reported failure end
// to end: a node whose Column plus Length runs past the end of the line it
// points at previously produced endColumn 17 on a 12 character line, which
// SonarQube rejects with "17 is not a valid line offset for pointer".
func TestParseSonarTextRangeCustomerRegression(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	const componentLine = "@Component({" // 12 characters, line 10 below
	source := "import { Component, inject, OnInit } from '@angular/core'\n" +
		"import { DomSanitizer } from '@angular/platform-browser'\n" +
		"import jwtDecode from 'jwt-decode'\n" +
		"import { TranslateModule } from '@ngx-translate/core'\n" +
		"import { MatCardModule } from '@angular/material/card'\n" +
		"\n\n\n\n" + // lines 6 to 9
		componentLine + "\n" + // line 10
		"  selector: 'app-last-login-ip',\n"

	fileName := writeSourceFile(t,
		dir,
		"cxone-sq-integration/juice-shop-master/frontend/src/app/last-login-ip/last-login-ip.component.ts",
		source,
	)

	index := newSonarLineIndex()
	node := &wrappers.ScanResultNode{
		FileName: fileName,
		Line:     10,
		Column:   12,
		Length:   6,
	}

	textRange := parseSonarTextRange(node, index)

	assert.NotNil(t, textRange)
	assert.Equal(t, uint(10), textRange.StartLine)
	assert.Equal(t, uint(11), textRange.StartColumn)
	assert.Equal(t, uint(12), textRange.EndColumn, "endColumn must be clamped to the 12 character line, not 17")
	assertValidSonarRange(t, textRange, 11, uint(len(componentLine)))
}

func TestParseSonarTextRangeFallsBackToLineLevel(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	index := newSonarLineIndex()

	t.Run("unreadable file keeps the line and emits no columns", func(t *testing.T) {
		node := &wrappers.ScanResultNode{FileName: "absent.ts", Line: 7, Column: 12, Length: 6}
		textRange := parseSonarTextRange(node, index)
		assert.NotNil(t, textRange)
		assert.Equal(t, uint(7), textRange.StartLine)
		assert.Zero(t, textRange.StartColumn, "columns are omitempty, so zero drops them from the report")
		assert.Zero(t, textRange.EndColumn)
	})

	t.Run("empty line emits no columns", func(t *testing.T) {
		fileName := writeSourceFile(t, dir, "blank.ts", "\nabc\n")
		node := &wrappers.ScanResultNode{FileName: fileName, Line: 1, Column: 3, Length: 4}
		textRange := parseSonarTextRange(node, index)
		assert.NotNil(t, textRange)
		assert.Equal(t, uint(1), textRange.StartLine)
		assert.Zero(t, textRange.StartColumn)
		assert.Zero(t, textRange.EndColumn)
	})
}

// TestParseSonarTextRangeOmitsRangeForMissingLine covers the minified asset case
// found on real scan data: the engine reported line 803 of a file that has only
// 82 lines, because its coordinates are against a transformed view of the
// source. Emitting that line at all aborts the SonarQube analysis, so the whole
// textRange must be dropped and the issue reported at file level.
func TestParseSonarTextRangeOmitsRangeForMissingLine(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fileName := writeSourceFile(t, dir, "assets/private/dat.gui.min.js", "var a=1\nvar b=2\n")
	index := newSonarLineIndex()

	node := &wrappers.ScanResultNode{FileName: fileName, Line: 803, Column: 5, Length: 4}
	textRange := parseSonarTextRange(node, index)

	assert.Nil(t, textRange, "textRange must be omitted when the line does not exist in the file")
}

// TestParseSonarAllLocationsAreValid walks a result with a primary and several
// secondary nodes, all deliberately overflowing, and asserts every emitted
// location satisfies SonarQube's invariant. A single violation anywhere in the
// report aborts the whole import, so secondary locations matter as much as the
// primary one.
func TestParseSonarAllLocationsAreValid(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	const line = "@Component({" // 12 characters
	fileName := writeSourceFile(t, dir, "app/widget.component.ts", line+"\n"+line+"\n")

	results := &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{
				Type: params.SastType,
				ScanResultData: wrappers.ScanResultData{
					QueryName: "Angular_Client_Stored_DOM_XSS",
					Nodes: []*wrappers.ScanResultNode{
						{FileName: fileName, Line: 1, Column: 12, Length: 6},
						{FileName: fileName, Line: 2, Column: 40, Length: 9},
						{FileName: fileName, Line: 1, Column: 1, Length: 0},
						{FileName: "missing.ts", Line: 3, Column: 5, Length: 4},
						// Line beyond end of a readable file: must be dropped.
						{FileName: fileName, Line: 803, Column: 5, Length: 4},
					},
				},
			},
		},
	}

	issues, _ := parseSonar(results)

	assert.Len(t, issues, 1)
	// Five nodes: one primary plus four secondaries, of which the node on line
	// 803 is dropped because SonarQube requires textRange on secondary
	// locations and no valid range exists for it.
	assert.Len(t, issues[0].SecondaryLocations, 3)

	all := append([]wrappers.SonarLocation{issues[0].PrimaryLocation}, issues[0].SecondaryLocations...)
	for i := range all {
		location := all[i]
		t.Run("location "+string(rune('A'+i)), func(t *testing.T) {
			if location.FilePath != fileName {
				// A file that is not on disk cannot be verified, so the engine
				// line is kept and only the columns are dropped.
				assert.NotNil(t, location.TextRange)
				assert.NotZero(t, location.TextRange.StartLine)
				assert.Zero(t, location.TextRange.StartColumn)
				assert.Zero(t, location.TextRange.EndColumn)
				return
			}
			assertValidSonarRange(t, location.TextRange, 2, uint(len(line)))
		})
	}

	// No secondary location may carry a nil textRange: SonarQube rejects the
	// whole report with "missing mandatory field 'textRange' in a secondary
	// location of the issue".
	for _, location := range issues[0].SecondaryLocations {
		assert.NotNil(t, location.TextRange, "secondary locations must always carry a textRange")
	}
}
