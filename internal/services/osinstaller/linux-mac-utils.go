//go:build linux || darwin

package osinstaller

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/checkmarx/ast-cli/internal/logger"
)

// dirDefault is the permission bits applied to directories created during extraction.
const dirDefault os.FileMode = 0755

// maxExtractBytes caps how many bytes a single tar entry may expand to,
// preventing decompression-bomb (tar-bomb) attacks.
const maxExtractBytes int64 = 500 * 1024 * 1024 // 500 MB

// UnzipOrExtractFiles Extracts all the files from the tar.gz file
func UnzipOrExtractFiles(installationConfiguration *InstallationConfiguration) error {
	logger.PrintIfVerbose("Extracting files in: " + installationConfiguration.WorkingDir())
	filePath := filepath.Join(installationConfiguration.WorkingDir(), installationConfiguration.FileName)

	gzipStream, err := os.Open(filePath)
	if err != nil {
		fmt.Println("error when open file ", filePath, err)
		return err
	}
	defer gzipStream.Close()

	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Println("ExtractTarGz: NewReader failed ", err)
		return err
	}
	defer uncompressedStream.Close()

	return extractFiles(installationConfiguration, tar.NewReader(uncompressedStream))
}

// safeJoin validates that name is a relative path and that the resolved
// destination stays inside workingDir, preventing path traversal attacks.
func safeJoin(workingDir, name string) (string, error) {
	if name == "" || name == "." {
		return "", fmt.Errorf("illegal file path (empty or dot): %s", name)
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("illegal file path (absolute): %s", name)
	}
	dst := filepath.Join(workingDir, name)
	cleanBase := filepath.Clean(workingDir) + string(os.PathSeparator)
	if !strings.HasPrefix(dst, cleanBase) {
		return "", fmt.Errorf("illegal file path (traversal): %s", name)
	}
	return dst, nil
}

func extractFiles(installationConfiguration *InstallationConfiguration, tarReader *tar.Reader) error {
	workingDir := installationConfiguration.WorkingDir()
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("ExtractTarGz: Next() failed: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			dst, err := safeJoin(workingDir, header.Name)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dst, dirDefault); err != nil {
				return fmt.Errorf("ExtractTarGz: Mkdir() failed: %w", err)
			}

		case tar.TypeReg:
			extractedFilePath, err := safeJoin(workingDir, header.Name)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(extractedFilePath), dirDefault); err != nil {
				return fmt.Errorf("ExtractTarGz: MkdirAll() failed: %w", err)
			}
			outFile, err := os.Create(extractedFilePath)
			if err != nil {
				return fmt.Errorf("ExtractTarGz: Create() failed: %w", err)
			}
			if _, err = io.Copy(outFile, io.LimitReader(tarReader, maxExtractBytes)); err != nil {
				_ = outFile.Close()
				return fmt.Errorf("ExtractTarGz: Copy() failed: %w", err)
			}
			if err = outFile.Close(); err != nil {
				return err
			}
			// Preserve only the executable bit from the archive entry; never grant world-write.
			mode := os.FileMode(0644)
			if header.FileInfo().Mode()&0111 != 0 {
				mode = 0755
			}
			if err = os.Chmod(extractedFilePath, mode); err != nil {
				return err
			}

		default:
			log.Printf("ExtractTarGz: unknown type: %v in %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

func ConfigureIndependentProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}
