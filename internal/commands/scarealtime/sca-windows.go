//go:build windows

package scarealtime

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
)

var Params = ScaRealTime{
	ExecutableFilePath:         filepath.Join(ScaResolverWorkingDir, "ScaResolver.exe"),
	HashFilePath:               filepath.Join(ScaResolverWorkingDir, "ScaResolver.zip.sha256sum"),
	SCAResolverDownloadURL:     "https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-win64.zip",
	SCAResolverHashDownloadURL: "https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-win64.zip.sha256sum",
	SCAResolverFileName:        "ScaResolver.zip",
	SCAResolverHashFileName:    "ScaResolver.zip.sha256sum",
}

// UnzipOrExtractFiles Extracts SCA Resolver files
func UnzipOrExtractFiles() error {
	logger.PrintIfVerbose("Unzipping files in:  " + ScaResolverWorkingDir)
	r, err := zip.OpenReader(filepath.Join(ScaResolverWorkingDir, Params.SCAResolverFileName))
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(ScaResolverWorkingDir, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(ScaResolverWorkingDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			err := os.MkdirAll(path, f.Mode())
			if err != nil {
				return err
			}
		} else {
			err := os.MkdirAll(filepath.Dir(path), f.Mode())
			if err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err = f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
