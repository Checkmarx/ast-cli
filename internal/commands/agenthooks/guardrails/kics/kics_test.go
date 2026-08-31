//go:build !integration

package kics

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/stretchr/testify/assert"
)

// ── isSupportedByKICS ────────────────────────────────────────────────────────

func TestIsSupportedByKICS_TerraformFile(t *testing.T) {
	if !isSupportedByKICS("/project/main.tf") {
		t.Error("expected .tf to be supported")
	}
}

func TestIsSupportedByKICS_YamlFile(t *testing.T) {
	if !isSupportedByKICS("/project/k8s/deployment.yaml") {
		t.Error("expected .yaml to be supported")
	}
}

func TestIsSupportedByKICS_YmlFile(t *testing.T) {
	if !isSupportedByKICS("/project/compose.yml") {
		t.Error("expected .yml to be supported")
	}
}

func TestIsSupportedByKICS_JsonFile(t *testing.T) {
	if !isSupportedByKICS("/project/policy.json") {
		t.Error("expected .json to be supported")
	}
}

func TestIsSupportedByKICS_Dockerfile(t *testing.T) {
	if !isSupportedByKICS("/project/Dockerfile") {
		t.Error("expected Dockerfile to be supported")
	}
}

func TestIsSupportedByKICS_DockerfileUppercase(t *testing.T) {
	if !isSupportedByKICS("/project/DOCKERFILE") {
		t.Error("expected DOCKERFILE (case-insensitive) to be supported")
	}
}

func TestIsSupportedByKICS_AutoTfvars(t *testing.T) {
	if !isSupportedByKICS("/project/prod.auto.tfvars") {
		t.Error("expected .auto.tfvars to be supported")
	}
}

func TestIsSupportedByKICS_TerraformTfvars(t *testing.T) {
	if !isSupportedByKICS("/project/env.terraform.tfvars") {
		t.Error("expected .terraform.tfvars to be supported")
	}
}

func TestIsSupportedByKICS_GoFileNotSupported(t *testing.T) {
	if isSupportedByKICS("/project/main.go") {
		t.Error("expected .go to NOT be supported")
	}
}

func TestIsSupportedByKICS_TxtFileNotSupported(t *testing.T) {
	if isSupportedByKICS("/project/README.txt") {
		t.Error("expected .txt to NOT be supported")
	}
}

func TestIsSupportedByKICS_PyFileNotSupported(t *testing.T) {
	if isSupportedByKICS("/project/app.py") {
		t.Error("expected .py to NOT be supported")
	}
}

// ── ScanFileEdit ─────────────────────────────────────────────────────────────

func makeResult(title, similarityID, severity, description string, line int) iacrealtime.IacRealtimeResult {
	return iacrealtime.IacRealtimeResult{
		Title:        title,
		SimilarityID: similarityID,
		Severity:     severity,
		Description:  description,
		Locations:    []realtimeengine.Location{{Line: line}},
	}
}

func TestScanFileEdit_NewFileWithFinding_Blocked(t *testing.T) {
	finding := makeResult("Privileged Container", "sim123", "HIGH", "Container runs as privileged", 5)
	svc := NewScannerWithFunc(func(_, _ string) ([]iacrealtime.IacRealtimeResult, error) {
		return []iacrealtime.IacRealtimeResult{finding}, nil
	})

	ev := agenthooks.FileEditEvent{
		FilePath:  "/project/Dockerfile",
		SessionID: "test-sess",
		Changes:   []agenthooks.FileDiff{{Before: "", After: "FROM ubuntu\nUSER root\n"}},
	}

	blocked, reason, ctx, _ := ScanFileEdit(ev, svc)
	if !blocked {
		t.Fatal("expected edit to be blocked")
	}
	if reason == "" {
		t.Error("expected non-empty reason")
	}
	if ctx == "" {
		t.Error("expected non-empty context")
	}
	if !strings.Contains(reason, "KICS") {
		t.Errorf("reason should mention KICS, got: %q", reason)
	}
}

func TestScanFileEdit_EditWithNoNewFindings_NotBlocked(t *testing.T) {
	existingFinding := makeResult("Privileged Container", "sim123", "HIGH", "Container runs as privileged", 5)
	svc := NewScannerWithFunc(func(_, _ string) ([]iacrealtime.IacRealtimeResult, error) {
		// Both original and new have the same finding — delta is empty
		return []iacrealtime.IacRealtimeResult{existingFinding}, nil
	})

	dir := t.TempDir()
	filePath := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(filePath, []byte("FROM ubuntu\nUSER root\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ev := agenthooks.FileEditEvent{
		FilePath:  filePath,
		SessionID: "test-sess",
		Changes:   []agenthooks.FileDiff{{Before: "FROM ubuntu", After: "FROM ubuntu:22.04"}},
	}

	blocked, _, _, _ := ScanFileEdit(ev, svc)
	if blocked {
		t.Fatal("expected edit to NOT be blocked when no new findings")
	}
}

func TestScanFileEdit_ScanError_FailOpen(t *testing.T) {
	svc := NewScannerWithFunc(func(_, _ string) ([]iacrealtime.IacRealtimeResult, error) {
		return nil, fmt.Errorf("docker daemon not running")
	})

	ev := agenthooks.FileEditEvent{
		FilePath:  "/project/main.tf",
		SessionID: "test-sess",
		Changes:   []agenthooks.FileDiff{{Before: "", After: "resource \"aws_s3_bucket\" \"bad\" {}"}},
	}

	blocked, _, _, _ := ScanFileEdit(ev, svc)
	if blocked {
		t.Fatal("expected fail-open (not blocked) on scan error")
	}
}

func TestScanFileEdit_UnsupportedFile_NotBlocked(t *testing.T) {
	svc := NewScannerWithFunc(func(_, _ string) ([]iacrealtime.IacRealtimeResult, error) {
		t.Error("scan should not be called for unsupported file types")
		return nil, nil
	})

	ev := agenthooks.FileEditEvent{
		FilePath:  "/project/main.go",
		SessionID: "test-sess",
		Changes:   []agenthooks.FileDiff{{Before: "", After: "package main"}},
	}

	blocked, _, _, _ := ScanFileEdit(ev, svc)
	if blocked {
		t.Fatal("expected NOT blocked for unsupported file")
	}
}

func TestScanFileEdit_EmptyNewContent_NotBlocked(t *testing.T) {
	svc := NewScannerWithFunc(func(_, _ string) ([]iacrealtime.IacRealtimeResult, error) {
		return nil, nil
	})

	ev := agenthooks.FileEditEvent{
		FilePath:  "/project/main.tf",
		SessionID: "test-sess",
		Changes:   []agenthooks.FileDiff{{Before: "", After: ""}},
	}

	blocked, _, _, _ := ScanFileEdit(ev, svc)
	if blocked {
		t.Fatal("expected NOT blocked for empty content")
	}
}

// A scan that could not run must say so — a silent fail-open is
// indistinguishable from a clean scan, which is how unscanned IaC ships.
func TestScanFileEdit_EngineDownProducesNote(t *testing.T) {
	svc := NewScannerWithFunc(func(path, ignoreFilePath string) ([]iacrealtime.IacRealtimeResult, error) {
		return nil, errors.New("container engine 'docker' is installed but not running")
	})
	ev := agenthooks.FileEditEvent{
		FilePath: filepath.Join(t.TempDir(), "main.tf"),
		Changes:  []agenthooks.FileDiff{{Before: "", After: "resource \"aws_s3_bucket\" \"b\" {}"}},
	}

	blocked, _, _, note := ScanFileEdit(ev, svc)

	assert.False(t, blocked, "must fail open, not block the edit")
	assert.Contains(t, note, "main.tf")
	assert.Contains(t, note, "not running")
}
