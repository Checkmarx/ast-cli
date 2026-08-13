package errorconstants

import (
	"strings"
	"testing"
)

func TestNewRealtimeEngineError(t *testing.T) {
	e := NewRealtimeEngineError("file path is required")
	if e == nil {
		t.Fatal("expected non-nil RealtimeEngineError")
	}
	if e.Message != "file path is required" {
		t.Errorf("Message = %q, want %q", e.Message, "file path is required")
	}
}

func TestRealtimeEngineError_Error(t *testing.T) {
	err := NewRealtimeEngineError("something broke").Error()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	got := err.Error()
	if !strings.Contains(got, "realtime engine error:") {
		t.Errorf("got %q, want format prefix", got)
	}
	if !strings.Contains(got, "something broke") {
		t.Errorf("got %q, want message body", got)
	}
}

func TestRealtimeEngineError_FormatConstant(t *testing.T) {
	if !strings.Contains(RealtimeEngineErrFormat, "%s") {
		t.Fatalf("RealtimeEngineErrFormat should include %%s, got %q", RealtimeEngineErrFormat)
	}
}

func TestErrorConstants_NonEmpty(t *testing.T) {
	consts := []string{
		StatusUnauthorized,
		StatusForbidden,
		RedirectURLNotFound,
		HTTPMethodNotFound,
		StatusInternalServerError,
		ApplicationDoesntExistOrNoPermission,
		ImportFilePathIsRequired,
		ProjectNameIsRequired,
		ProjectNotExists,
		ScanIDRequired,
		FailedToGetApplication,
		SarifInvalidFileExtension,
		ImportSarifFileError,
		NoASCALicense,
		NoPermissionToUpdateApplication,
		FailedToUpdateApplication,
		ApplicationNotFound,
		ErrMissingAIFeatureLicense,
		FileExtensionIsRequired,
		RealtimeEngineNotAvailable,
		RealtimeEngineFilePathRequired,
	}
	for _, c := range consts {
		if strings.TrimSpace(c) == "" {
			t.Error("expected non-empty error constant")
		}
	}
}

func TestImportSarifFileErrorMessageWithMessage_Format(t *testing.T) {
	if !strings.Contains(ImportSarifFileErrorMessageWithMessage, "%d") ||
		!strings.Contains(ImportSarifFileErrorMessageWithMessage, "%s") {
		t.Fatalf("expected format verbs in %q", ImportSarifFileErrorMessageWithMessage)
	}
}

func TestFailedUploadFileMsg_Format(t *testing.T) {
	if !strings.Contains(FailedUploadFileMsgWithDomain, "%s") {
		t.Fatalf("expected %%s in %q", FailedUploadFileMsgWithDomain)
	}
	if !strings.Contains(FailedUploadFileMsgWithURL, "%s") {
		t.Fatalf("expected %%s in %q", FailedUploadFileMsgWithURL)
	}
}
