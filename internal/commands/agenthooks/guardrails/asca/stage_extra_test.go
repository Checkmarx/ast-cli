//go:build !integration

package asca

import (
	"testing"
	"unicode/utf8"
)

func TestAsciiSafe_AllASCIIUnchanged(t *testing.T) {
	got := asciiSafe("package main // clean ascii comment")
	want := "package main // clean ascii comment"
	if got != want {
		t.Errorf("asciiSafe() = %q, want %q", got, want)
	}
}

func TestAsciiSafe_NonASCIIReplacedWithSpace(t *testing.T) {
	in := "// café comment"
	got := asciiSafe(in)
	want := "// caf  comment"
	if got != want {
		t.Errorf("asciiSafe() = %q, want %q", got, want)
	}
	if !utf8.ValidString(got) {
		t.Errorf("asciiSafe() produced invalid utf8: %q", got)
	}
	for _, r := range got {
		if r > maxASCIICodePoint {
			t.Errorf("asciiSafe() left a non-ASCII rune %q in %q", r, got)
		}
	}
}

func TestAsciiSafe_PreservesLineStructure(t *testing.T) {
	in := "line1\nline2 with é\nline3"
	got := asciiSafe(in)
	wantLines := 3
	lines := 1
	for _, r := range got {
		if r == '\n' {
			lines++
		}
	}
	if lines != wantLines {
		t.Errorf("asciiSafe() changed line count: got %d lines, want %d", lines, wantLines)
	}
}

func TestAsciiSafe_EmptyString(t *testing.T) {
	if got := asciiSafe(""); got != "" {
		t.Errorf("asciiSafe(\"\") = %q, want empty", got)
	}
}
