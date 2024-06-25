package osinstaller

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

// downloadFile Downloads a file from url path
func downloadFile(downloadURLPath, filePath string) error {
	_, fileName := filepath.Split(filePath)
	logger.PrintIfVerbose("Downloading " + fileName + " from: " + downloadURLPath)

	response, err := wrappers.SendHTTPRequestByFullURL(http.MethodGet, downloadURLPath, http.NoBody, false, 0, "", true)
	if err != nil {
		return errors.Errorf("Invoking HTTP request to download file failed - %s", err.Error())
	}
	defer func() {
		_ = response.Body.Close()
	}()

	zipFile, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Error creating zip file: %s", err)
		return err
	}
	defer func() {
		_ = zipFile.Close()
	}()

	// Write the body to file
	_, err = io.Copy(zipFile, response.Body)
	if err != nil {
		fmt.Printf("Error writing the body response to zip file: %s", err)
		return err
	}

	return nil
}

// InstallOrUpgrade Checks the version according to the hash file,
// downloads the RealTime installation if the version is not up-to-date,
// Extracts the RealTime installation according to the operating system type
func InstallOrUpgrade(installationConfiguration *InstallationConfiguration) error {
	logger.PrintIfVerbose("Handling RealTime Installation...")
	if downloadNotNeeded(installationConfiguration) {
		logger.PrintIfVerbose("RealTime installation already exists and is up to date. Skipping download.")
		return nil
	}

	// Create temporary working directory if not exists
	err := createWorkingDirectory(installationConfiguration)
	if err != nil {
		return err
	}

	// Download RealTime installation
	err = downloadFile(installationConfiguration.DownloadURL, filepath.Join(installationConfiguration.WorkingDir(), installationConfiguration.FileName))
	if err != nil {
		return err
	}

	// Download hash file
	err = downloadHashFile(installationConfiguration.HashDownloadURL, installationConfiguration.HashFilePath())
	if err != nil {
		return err
	}

	// Unzip or extract downloaded zip depending on which OS is running
	err = UnzipOrExtractFiles(installationConfiguration)
	if err != nil {
		return err
	}

	return nil
}

// createWorkingDirectory Creates a working directory to handle Realtime functionality
func createWorkingDirectory(installationConfiguration *InstallationConfiguration) error {
	logger.PrintIfVerbose("Creating temporary directory to handle Realtime...")
	err := os.MkdirAll(installationConfiguration.WorkingDir(), fs.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// downloadNotNeeded Checks if the installation is already available and if it is up-to-date
func downloadNotNeeded(installationConfiguration *InstallationConfiguration) bool {
	logger.PrintIfVerbose("Checking if RealTime installation already exists...")
	executableFileExists, _ := FileExists(installationConfiguration.ExecutableFilePath())

	if !executableFileExists {
		return false
	}

	logger.PrintIfVerbose("RealTime installation exists. Checking if it is the latest version...")

	isLastVersion, _ := isLastVersion(installationConfiguration.HashFilePath(), installationConfiguration.HashDownloadURL, installationConfiguration.HashFilePath())

	return isLastVersion
}

// isLastVersion Checks if the Installation is updated by comparing hashes
func isLastVersion(hashFilePath, hashURL, zipFileNameHash string) (bool, error) {
	existingHash, _ := getHashValue(hashFilePath)
	// Download hash file
	err := downloadHashFile(hashURL, zipFileNameHash)
	if err != nil {
		return false, err
	}
	currentHash, _ := getHashValue(hashFilePath)
	if !bytes.Equal(existingHash, currentHash) {
		logger.PrintIfVerbose("The RealTime installation is out of date.")
	}
	return bytes.Equal(existingHash, currentHash), nil
}

// FileExists Check if a file exists in a specific directory
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// getHashValue Gets the hash value of a file
func getHashValue(hashFilePath string) ([]byte, error) {
	f, err := os.Open(hashFilePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// downloadHashFile Downloads hash file
func downloadHashFile(hashURL, zipFileNameHash string) error {
	err := downloadFile(hashURL, zipFileNameHash)
	if err != nil {
		return err
	}

	return nil
}
