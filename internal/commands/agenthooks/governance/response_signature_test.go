package governance

import (
	"encoding/json"
	"testing"
)

func TestFailureText_SuccessfulCallIsNotAFailure(t *testing.T) {
	raw := json.RawMessage(`{"output":"file1\nfile2","exitCode":0}`)
	_, failed := failureText(raw)
	if failed {
		t.Error("exit code 0 must not be treated as a failure")
	}
}

func TestFailureText_NonZeroExitIsAFailure(t *testing.T) {
	raw := json.RawMessage(`{"output":"","error":"Access is denied.","exitCode":1}`)
	text, failed := failureText(raw)
	if !failed {
		t.Fatal("non-zero exit code must be treated as a failure")
	}
	if text == "" {
		t.Error("failure text must not be empty when an error field is present")
	}
}

func TestFailureText_ExplicitSuccessFalse(t *testing.T) {
	raw := json.RawMessage(`{"success":false,"message":"write refused"}`)
	text, failed := failureText(raw)
	if !failed || text != "write refused" {
		t.Errorf("text=%q failed=%v", text, failed)
	}
}

func TestFailureText_EmptyOrMissingResponse(t *testing.T) {
	if _, failed := failureText(nil); failed {
		t.Error("nil response must not be a failure")
	}
	if _, failed := failureText(json.RawMessage(`not json`)); failed {
		t.Error("unparseable response must not be a failure")
	}
}

func TestMatchBlockSignature_VendorSpecificIsDefinitive(t *testing.T) {
	name, conf, ok := matchBlockSignature("This operation was blocked: Windows Defender detected a threat")
	if !ok || name != "Windows Defender" || conf != "definitive" {
		t.Errorf("name=%q conf=%q ok=%v", name, conf, ok)
	}
}

func TestMatchBlockSignature_CaseInsensitive(t *testing.T) {
	if _, _, ok := matchBlockSignature("blocked by CROWDSTRIKE falcon sensor"); !ok {
		t.Error("matching must be case-insensitive")
	}
}

func TestMatchBlockSignature_GenericPhraseIsCandidate(t *testing.T) {
	name, conf, ok := matchBlockSignature("Request blocked by your organization's web filter")
	if !ok || conf != "candidate" {
		t.Errorf("name=%q conf=%q ok=%v, want a candidate-confidence match", name, conf, ok)
	}
}

func TestMatchBlockSignature_MundaneErrorsDoNotMatch(t *testing.T) {
	mundane := []string{
		"Access is denied.", // bare — could be any OS permission issue
		"Permission denied",
		"No such file or directory",
		"fatal: not a git repository",
		"npm ERR! code ENOENT",
	}
	for _, text := range mundane {
		if _, _, ok := matchBlockSignature(text); ok {
			t.Errorf("matchBlockSignature(%q) matched, want no match — this is ordinary failure text", text)
		}
	}
}

func TestMatchBlockSignature_EmptyText(t *testing.T) {
	if _, _, ok := matchBlockSignature(""); ok {
		t.Error("empty text must never match")
	}
}

func TestDetectResponseSignatureBlock_OnlyRunsForShellAndFilesystem(t *testing.T) {
	raw := json.RawMessage(`{"error":"blocked by CrowdStrike Falcon","exitCode":1}`)

	if _, _, _, ok := detectResponseSignatureBlock("shell", raw); !ok {
		t.Error("shell category must be checked")
	}
	if _, _, _, ok := detectResponseSignatureBlock("filesystem", raw); !ok {
		t.Error("filesystem category must be checked")
	}
	if _, _, _, ok := detectResponseSignatureBlock("mcp", raw); ok {
		t.Error("mcp category must not be checked — a failed API call is not a security block")
	}
	if _, _, _, ok := detectResponseSignatureBlock("web_search", raw); ok {
		t.Error("web_search category must not be checked")
	}
}

func TestDetectResponseSignatureBlock_SuccessfulCallNeverMatches(t *testing.T) {
	// Even if the output happens to mention a vendor name, a successful call
	// (exit 0, no error) must never be flagged as blocked by it.
	raw := json.RawMessage(`{"output":"scanning with mcafee complete, 0 threats found","exitCode":0}`)
	if _, _, _, ok := detectResponseSignatureBlock("shell", raw); ok {
		t.Error("a successful call must never be scanned for a block signature")
	}
}

func TestDetectResponseSignatureBlock_FullMatch(t *testing.T) {
	raw := json.RawMessage(`{"error":"Blocked by Windows Defender: potentially unwanted software","exitCode":1}`)
	name, reason, confidence, ok := detectResponseSignatureBlock("shell", raw)
	if !ok {
		t.Fatal("expected a match")
	}
	if name != "Windows Defender" {
		t.Errorf("name=%q", name)
	}
	if confidence != "definitive" {
		t.Errorf("confidence=%q", confidence)
	}
	if reason == "" {
		t.Error("reason must carry the matched text")
	}
}

func TestFirstLines(t *testing.T) {
	got := firstLines("line one\nline two\nline three\nline four", 2)
	want := "line one\nline two"
	if got != want {
		t.Errorf("firstLines() = %q, want %q", got, want)
	}
}
