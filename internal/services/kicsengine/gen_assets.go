//go:build ignore

// Command gen_assets builds the embedded KICS asset archive.
//
// It reads the assets straight out of the KICS module version pinned in go.mod, so the
// embedded queries can never disagree with the KICS code we link against. Bumping KICS is
// therefore `go get github.com/Checkmarx/kics/v2@<version>` followed by `go generate ./...`.
//
// Which files get included is defined once in the assetspec package, shared with the drift
// test that guards this archive - so the guard cannot silently agree with a stale copy of
// the rule it is checking.
package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/checkmarx/ast-cli/internal/services/kicsengine/assetspec"
)

const (
	kicsModule = "github.com/Checkmarx/kics/v2"
	outputFile = "assets/kics-assets.tar.gz"

	archiveFileMode = 0o644
	archiveDirMode  = 0o755
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen_assets: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	moduleDir, err := kicsModuleDir()
	if err != nil {
		return err
	}

	files, err := collect(moduleDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no assets found under %s", moduleDir)
	}

	if err := writeArchive(moduleDir, files); err != nil {
		return err
	}

	fmt.Printf("gen_assets: wrote %s from %s (%d files)\n", outputFile, moduleDir, len(files))
	return nil
}

// kicsModuleDir asks the go tool where the pinned KICS module is unpacked. Resolving it this
// way rather than composing a GOMODCACHE path avoids the module cache's case-escaping rules.
func kicsModuleDir() (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", kicsModule).Output()
	if err != nil {
		return "", fmt.Errorf("locating %s (run `go mod download` first): %w", kicsModule, err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("%s has no local directory", kicsModule)
	}
	return dir, nil
}

// collect returns slash-separated paths relative to moduleDir, sorted so the archive is
// byte-identical across runs and platforms.
func collect(moduleDir string) ([]string, error) {
	var files []string

	for _, tree := range assetspec.Trees() {
		root := filepath.Join(moduleDir, filepath.FromSlash(tree))
		err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !assetspec.Include(tree, info.Name()) {
				return nil
			}
			rel, relErr := filepath.Rel(moduleDir, p)
			if relErr != nil {
				return relErr
			}
			files = append(files, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking %s: %w", tree, err)
		}
	}

	sort.Strings(files)
	return files, nil
}

// writeArchive emits a deterministic tar.gz: sorted entries, zeroed timestamps and fixed
// modes, so regenerating from the same KICS version reproduces the file bit for bit and CI
// can diff it.
func writeArchive(moduleDir string, files []string) error {
	if err := os.MkdirAll(filepath.Dir(outputFile), archiveDirMode); err != nil {
		return err
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	gz, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	tw := tar.NewWriter(gz)

	for _, rel := range files {
		if err := addFile(tw, moduleDir, rel); err != nil {
			return fmt.Errorf("adding %s: %w", rel, err)
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return out.Close()
}

func addFile(tw *tar.Writer, moduleDir, rel string) error {
	src, err := os.Open(filepath.Join(moduleDir, filepath.FromSlash(rel)))
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	// Name is already slash-separated and relative; path.Clean guards against any oddity
	// that would otherwise land outside the extraction root.
	if err := tw.WriteHeader(&tar.Header{
		Name:     path.Clean(rel),
		Mode:     archiveFileMode,
		Size:     info.Size(),
		Typeflag: tar.TypeReg,
		Format:   tar.FormatPAX,
	}); err != nil {
		return err
	}

	_, err = io.Copy(tw, src)
	return err
}
