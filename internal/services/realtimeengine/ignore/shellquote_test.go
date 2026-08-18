package ignore

import (
	"runtime"
	"testing"
)

func TestQuoteDataFlag_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix quoting on windows host")
	}
	got := QuoteDataFlag([]byte(`{"FileName":"a.py","Line":1,"RuleID":2}`))
	want := `'{"FileName":"a.py","Line":1,"RuleID":2}'`
	if got != want {
		t.Fatalf("QuoteDataFlag() = %q, want %q", got, want)
	}
}

func TestQuoteDataFlag_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows quoting")
	}
	got := QuoteDataFlag([]byte(`{"FileName":"a.py","Line":1,"RuleID":2}`))
	want := `'{\"FileName\":\"a.py\",\"Line\":1,\"RuleID\":2}'`
	if got != want {
		t.Fatalf("QuoteDataFlag() = %q, want %q", got, want)
	}
}
