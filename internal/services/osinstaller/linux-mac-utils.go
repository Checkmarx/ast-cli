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

const dirDefault int = 0755

// UnzipOrExtractFiles Extracts all the files from the tar.gz file
func UnzipOrExtractFiles(installationConfiguration *InstallationConfiguration) error {
	logger.PrintIfVerbose("Extracting files in: " + installationConfiguration.WorkingDir())
	filePath := filepath.Join(installationConfiguration.WorkingDir(), installationConfiguration.FileName)
	gzipStream, err := os.Open(filePath)
	if err != nil {
		fmt.Println("error when open file ", filePath, err)
		return err
	}
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Println("ExtractTarGz: NewReader failed ", err)
		return err
	}

	tarReader := tar.NewReader(uncompressedStream)

	err = extractFiles(installationConfiguration, tarReader)
	if err != nil {
		return err
	}
	return nil
}

func extractFiles(installationConfiguration *InstallationConfiguration, tarReader *tar.Reader) error {
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
			if err := os.Mkdir(header.Name, os.FileMode(dirDefault)); err != nil {
				log.Fatalf("ExtractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			extractedFilePath := filepath.Join(installationConfiguration.WorkingDir(), header.Name)
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
