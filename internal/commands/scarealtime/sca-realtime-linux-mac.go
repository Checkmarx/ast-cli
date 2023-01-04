//go:build linux || darwin

package scarealtime

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/checkmarx/ast-cli/internal/logger"
)

var linuxMacSCARealTime = ScaRealTime{
	ExecutableFilePath:         filepath.Join(scaResolverWorkingDir, "ScaResolver"),
	HashFilePath:               filepath.Join(scaResolverWorkingDir, "ScaResolver.tar.gz.sha256sum"),
	SCAResolverDownloadURL:     "https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz",
	SCAResolverHashDownloadURL: "https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz.sha256sum",
	SCAResolverFileName:        "ScaResolver.tar.gz",
	SCAResolverHashFileName:    "ScaResolver.tar.gz.sha256sum",
}

// getScaResolver Gets SCA Resolver file path to run SCA Realtime
func getScaResolver() (string, error) {
	if runtime.GOOS == "darwin" {
		linuxMacSCARealTime.SCAResolverDownloadURL = "https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-macos64.tar.gz"
		linuxMacSCARealTime.SCAResolverHashDownloadURL = "https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-macos64.tar.gz.sha256sum"
	}
	err := downloadSCAResolverAndHashFileIfNeeded(&linuxMacSCARealTime)
	if err != nil {
		return "", err
	}

	return linuxMacSCARealTime.ExecutableFilePath, nil
}

// unzipOrExtractFiles Extracts SCA Resolver files
func unzipOrExtractFiles() error {
	logger.PrintIfVerbose("Extracting files in: " + scaResolverWorkingDir)
	gzipStream, err := os.Open(filepath.Join(scaResolverWorkingDir, linuxMacSCARealTime.SCAResolverFileName))
	if err != nil {
		fmt.Println("error")
	}
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("ExtractTarGz: NewReader failed")
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
			if err := os.Mkdir(header.Name, 0755); err != nil { //nolint:gomnd
				log.Fatalf("ExtractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			extractedFilePath := filepath.Join(scaResolverWorkingDir, header.Name)
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
