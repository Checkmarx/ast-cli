//go:build !integration

package asca

import "testing"

func TestNormLF_CRLFNormalized(t *testing.T) {
	got := normLF("line1\r\nline2\r\nline3")
	want := "line1\nline2\nline3"
	if got != want {
		t.Errorf("normLF() = %q, want %q", got, want)
	}
}

func TestNormLF_BareCRNormalized(t *testing.T) {
	got := normLF("line1\rline2\rline3")
	want := "line1\nline2\nline3"
	if got != want {
		t.Errorf("normLF() = %q, want %q", got, want)
	}
}

func TestNormLF_AlreadyLFUnchanged(t *testing.T) {
	got := normLF("line1\nline2\nline3")
	want := "line1\nline2\nline3"
	if got != want {
		t.Errorf("normLF() = %q, want %q", got, want)
	}
}
