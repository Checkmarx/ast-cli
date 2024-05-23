//go:build linux || darwin

package osinstaller

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/checkmarx/ast-cli/internal/logger"
)

var DirDefault = os.FileMode(0755)

// UnzipOrExtractFiles Extracts SCA Resolver files
func UnzipOrExtractFiles(installableRealtime *InstallableRealTime) error {
	logger.PrintIfVerbose("Extracting files in: " + installableRealtime.WorkingDir())
	gzipStream, err := os.Open(filepath.Join(installableRealtime.WorkingDir(), installableRealtime.FileName))
	if err != nil {
		fmt.Println("error")
	}
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("ExtractTarGz: NewReader failed", err)
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("ExtractTarGz: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(header.Name, DirDefault); err != nil {
				log.Fatalf("ExtractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			//TODO: delete
			if header.Name == "dist/cx-mac-universal_darwin_all/cx.dmg" {
				// Skip the cx.dmg file
				continue
			}
			extractedFilePath := filepath.Join(installableRealtime.WorkingDir(), header.Name)
			outFile, err := os.Create(extractedFilePath)
			if err != nil {
				log.Fatalf("ExtractTarGz: Create() failed: %s", err.Error())
			}
			if _, err = io.Copy(outFile, tarReader); err != nil {
				log.Fatalf("ExtractTarGz: Copy() failed: %s", err.Error())
			}
			err = outFile.Close()
			if err != nil {
				return err
			}
			err = os.Chmod(extractedFilePath, fs.ModePerm)
			if err != nil {
				return err
			}
		default:
			log.Fatalf(
				"ExtractTarGz: uknown type: %v in %s",
				header.Typeflag,
				header.Name)
		}
	}

	return nil
}
