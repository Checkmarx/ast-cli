//go:build !integration

package kics

import (
	"os"
	"path/filepath"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/stretchr/testify/assert"
)

const wantNormalizedLines = "line1\nline2\nline3"

func TestNormLF_CRLFNormalized(t *testing.T) {
	assert.Equal(t, wantNormalizedLines, normLF("line1\r\nline2\r\nline3"))
}

func TestNormLF_BareCRNormalized(t *testing.T) {
	assert.Equal(t, wantNormalizedLines, normLF("line1\rline2\rline3"))
}

func TestNormLF_AlreadyLFUnchanged(t *testing.T) {
	assert.Equal(t, wantNormalizedLines, normLF("line1\nline2\nline3"))
}

func TestProposedContent_FullFileWrite(t *testing.T) {
	newContent, _, err := proposedContent("/nonexistent/main.tf", []agenthooks.FileDiff{
		{Before: "", After: "resource \"aws_s3_bucket\" \"b\" {}"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "resource \"aws_s3_bucket\" \"b\" {}", newContent)
}

func TestProposedContent_StringReplaceEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.tf")
	assert.NoError(t, os.WriteFile(path, []byte("bucket = \"a\"\n"), 0o600))

	newContent, origContent, err := proposedContent(path, []agenthooks.FileDiff{
		{Before: "bucket = \"a\"", After: "bucket = \"b\""},
	})
	assert.NoError(t, err)
	assert.Equal(t, "bucket = \"a\"\n", origContent)
	assert.Equal(t, "bucket = \"b\"\n", newContent)
}

func TestProposedContent_MissingBeforeFailsOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.tf")
	assert.NoError(t, os.WriteFile(path, []byte("a = 1\n"), 0o600))

	newContent, origContent, err := proposedContent(path, []agenthooks.FileDiff{
		{Before: "NOTHERE", After: "replacement"},
	})
	assert.NoError(t, err)
	assert.Equal(t, origContent, newContent)
}

func TestProposedContent_CRLFDiskWithLFDiff_AppliesEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.tf")
	diskContent := "resource \"aws_s3_bucket\" \"public_bucket\" {\r\n  bucket = \"b\"\r\n}\r\n"
	assert.NoError(t, os.WriteFile(path, []byte(diskContent), 0o600))

	newContent, origContent, err := proposedContent(path, []agenthooks.FileDiff{
		{
			Before: "resource \"aws_s3_bucket\" \"public_bucket\" {\n  bucket = \"b\"\n}",
			After:  "resource \"aws_s3_bucket\" \"public_bucket\" {\n  bucket = \"b\"\n  acl    = \"public-read\"\n}",
		},
	})

	assert.NoError(t, err)
	assert.NotEqual(t, origContent, newContent)
	assert.Contains(t, newContent, "acl    = \"public-read\"")
}
