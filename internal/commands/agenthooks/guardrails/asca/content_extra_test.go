//go:build !integration

package asca

import "testing"

const wantNormalizedLines = "line1\nline2\nline3"

func TestNormLF_CRLFNormalized(t *testing.T) {
	got := normLF("line1\r\nline2\r\nline3")
	if got != wantNormalizedLines {
		t.Errorf("normLF() = %q, want %q", got, wantNormalizedLines)
	}
}

func TestNormLF_BareCRNormalized(t *testing.T) {
	got := normLF("line1\rline2\rline3")
	if got != wantNormalizedLines {
		t.Errorf("normLF() = %q, want %q", got, wantNormalizedLines)
	}
}

func TestNormLF_AlreadyLFUnchanged(t *testing.T) {
	got := normLF("line1\nline2\nline3")
	if got != wantNormalizedLines {
		t.Errorf("normLF() = %q, want %q", got, wantNormalizedLines)
	}
}
