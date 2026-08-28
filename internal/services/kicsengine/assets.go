// Package kicsengine runs KICS in-process, with its query assets compiled into the CLI.
//
// It replaces the two previous backends for IaC scanning - a KICS container image and a
// KICS binary downloaded from GitHub releases - so an IaC scan needs no container runtime,
// no network access and no first-run install.
package kicsengine

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/services/kicsengine/assetspec"
	"github.com/pkg/errors"

	_ "embed"
)

//go:generate go run gen_assets.go

// assetArchive holds the KICS query set, Rego libraries and similarity-ID transition files,
// generated from the KICS version pinned in go.mod. See gen_assets.go.
//
//go:embed assets/kics-assets.tar.gz
var assetArchive []byte

const (
	// kicsModulePath identifies the pinned KICS module; the asset drift guard uses it to
	// locate that module in the go module cache.
	kicsModulePath = "github.com/Checkmarx/kics/v2"

	// installDirName sits under the CLI's configuration directory, matching the layout the
	// previous downloading installer used.
	installDirName = "CxKics"

	// completeMarker is written last, so a half-extracted directory is never mistaken for a
	// usable one.
	completeMarker = ".complete"

	assetsDirName     = assetspec.Root
	queriesDirName    = assetspec.QueriesDir
	librariesDirName  = assetspec.LibrariesDir
	transitionDirName = assetspec.TransitionDir

	// revisionLen keeps the cache directory name short while staying collision-free in
	// practice; it is a content address, not a security boundary.
	revisionLen = 12

	extractDirMode  = 0o700
	extractFileMode = 0o600

	// maxAssetBytes caps how much a single archive entry may expand to, guarding against a
	// decompression bomb if the embedded archive were ever replaced.
	maxAssetBytes = 64 << 20
)

var (
	extractOnce sync.Once
	extractRoot string
	extractErr  error
)

// AssetsRoot extracts the embedded KICS assets on first use and returns the directory that
// contains them. Subsequent calls reuse the extracted copy.
//
// KICS resolves its query and library paths from the filesystem, so the assets have to land
// on disk even though they ship inside the binary.
func AssetsRoot() (string, error) {
	extractOnce.Do(func() { extractRoot, extractErr = extractAssets() })
	return extractRoot, extractErr
}

// revision is a content address of the embedded assets. Keying the cache directory on it means
// a rebuilt archive always extracts to a fresh directory instead of being masked by a stale
// one, whether or not the KICS version changed.
//
// It is deliberately the hash alone. Prefixing the KICS version read from build info looked
// friendlier, but debug.ReadBuildInfo records no dependency versions under "go test", so the
// test binary and the real binary resolved to different cache directories - which quietly
// pointed the asset drift guard at a tree other than the one being tested.
func revision() string {
	sum := sha256.Sum256(assetArchive)
	return hex.EncodeToString(sum[:])[:revisionLen]
}

func installRoot() (string, error) {
	base, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "resolving home directory for KICS assets")
	}
	return filepath.Join(base, ".checkmarx", installDirName, revision()), nil
}

func extractAssets() (string, error) {
	root, err := installRoot()
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(filepath.Join(root, completeMarker)); statErr == nil {
		return root, nil
	}

	logger.PrintIfVerbose("Extracting embedded KICS assets to " + root)

	parent := filepath.Dir(root)
	if err := os.MkdirAll(parent, extractDirMode); err != nil {
		return "", errors.Wrap(err, "creating KICS asset directory")
	}

	// Extract to a sibling directory and rename into place, so a concurrent CLI process
	// never observes a partially written asset tree.
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(root)+".tmp")
	if err != nil {
		return "", errors.Wrap(err, "creating staging directory for KICS assets")
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if err := unpack(assetArchive, staging); err != nil {
		return "", errors.Wrap(err, "unpacking embedded KICS assets")
	}

	marker, err := os.Create(filepath.Join(staging, completeMarker))
	if err != nil {
		return "", errors.Wrap(err, "writing KICS asset marker")
	}
	if err := marker.Close(); err != nil {
		return "", errors.Wrap(err, "writing KICS asset marker")
	}

	if err := os.Rename(staging, root); err != nil {
		// Another process may have finished first; its copy is equivalent because the
		// directory name is a content address of the same archive.
		if _, statErr := os.Stat(filepath.Join(root, completeMarker)); statErr == nil {
			return root, nil
		}
		return "", errors.Wrap(err, "publishing KICS assets")
	}

	return root, nil
}

func unpack(archive []byte, destDir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if err := writeEntry(tr, destDir, header); err != nil {
			return err
		}
	}
}

func writeEntry(tr io.Reader, destDir string, header *tar.Header) error {
	target, err := safeJoin(destDir, header.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), extractDirMode); err != nil {
		return err
	}

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, extractFileMode)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, io.LimitReader(tr, maxAssetBytes)); err != nil {
		return err
	}
	return out.Close()
}

// safeJoin resolves an archive entry name under destDir, rejecting anything that would escape
// it. The archive is generated and embedded by us, but a path traversal here would write
// outside the cache, so the check is cheap insurance.
func safeJoin(destDir, name string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", errors.Errorf("illegal asset path in archive: %s", name)
	}
	target := filepath.Join(destDir, cleaned)
	if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return "", errors.Errorf("illegal asset path in archive: %s", name)
	}
	return target, nil
}
