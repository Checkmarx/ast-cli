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
	"strings"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	grpcs "github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/pkg/errors"
)

type NewSuccessfulInstallation bool

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
func InstallOrUpgrade(installationConfiguration *InstallationConfiguration, ascaWrapper grpcs.AscaWrapper) (NewSuccessfulInstallation, error) {
	logger.PrintIfVerbose("Handling RealTime Installation...")
	if downloadNotNeeded(installationConfiguration) {
		logger.PrintIfVerbose("RealTime installation already exists and is up to date. Skipping download.")
		return false, nil
	}

	// Create temporary working directory if not exists
	err := createWorkingDirectory(installationConfiguration)
	if err != nil {
		return false, err
	}

	// Download RealTime installation
	err = downloadFile(installationConfiguration.DownloadURL, filepath.Join(installationConfiguration.WorkingDir(), installationConfiguration.FileName))
	if err != nil {
		return false, err
	}

	// Hash file serves different purposes: version check for Vorpal, both version check and verification for SCA
	err = downloadHashFile(installationConfiguration.HashDownloadURL, installationConfiguration.HashFilePath())
	if err != nil {
		return false, err
	}

	// Must shut down service before replacement to release file locks
	if ascaWrapper != nil {
		shutDownAndWait(ascaWrapper)
	}

	checksumPath, needsArchiveChecksumDownload, err := installationConfiguration.resolveArchiveChecksumVerification()
	if err != nil {
		return false, err
	}
	if needsArchiveChecksumDownload {
		err = downloadFile(installationConfiguration.ArchiveChecksumDownloadURL, checksumPath)
		if err != nil {
			return false, err
		}
	}
	if checksumPath != "" {
		err = verifyArchiveAgainstSHA256SumFile(installationConfiguration.BinaryFilePath(), checksumPath, installationConfiguration.DownloadURL)
		if err != nil {
			logger.PrintIfVerbose("Removing potentially compromised archive due to checksum verification failure")
			_ = os.Remove(installationConfiguration.BinaryFilePath())
			return false, errors.Errorf("Archive integrity verification failed for %s: Checksum verification failed - archive may have been compromised", installationConfiguration.ExecutableFile)
		}
	} else {
		logger.PrintIfVerbose("Skipping archive checksum verification (no sha256sum source configured for this installation)")
	}

	// Unzip or extract downloaded zip depending on which OS is running
	err = UnzipOrExtractFiles(installationConfiguration)
	if err != nil {
		return false, err
	}

	return true, nil
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

// shutDownAndWait sends a shutdown signal and polls until the service is no longer reachable,
// ensuring the process has released its file handles before the caller replaces the binary.
func shutDownAndWait(ascaWrapper grpcs.AscaWrapper) {
	const (
		maxAttempts  = 20
		pollInterval = 500 * time.Millisecond
	)

	logger.PrintIfVerbose("Shutting down Vorpal service before replacing binary...")
	_ = ascaWrapper.ShutDown()

	port := ascaWrapper.GetPort()
	for i := 0; i < maxAttempts; i++ {
		// ConfigurePort resets the cached 'serving' flag, forcing a live connection attempt.
		ascaWrapper.ConfigurePort(port)
		if err := ascaWrapper.HealthCheck(); err != nil {
			logger.PrintIfVerbose("Vorpal service has stopped.")
			return
		}
		time.Sleep(pollInterval)
	}
	logger.PrintIfVerbose("Timed out waiting for Vorpal service to stop; proceeding anyway.")
}

// verifyArchiveAgainstSHA256SumFile checks that archivePath matches the digest in a GNU sha256sum-style file.
// For Vorpal: searches using the platform-specific filename from downloadURL.
// Supports single-line format (one checksum) or multi-line format (searches for matching filename).
func verifyArchiveAgainstSHA256SumFile(archivePath, sha256SumFilePath, downloadURL string) error {
	logger.PrintIfVerbose("Verifying downloaded archive against sha256sum checksum")

	content, err := os.ReadFile(sha256SumFilePath)
	if err != nil {
		return errors.Errorf("Failed to read checksum file: %s", err.Error())
	}

	fileContent := strings.TrimSpace(string(content))
	if fileContent == "" {
		return errors.New("Checksum file is empty")
	}

	// Extract the actual platform-specific filename from downloadURL
	_, downloadFileName := filepath.Split(downloadURL)
	logger.PrintIfVerbose("Searching checksum file for: " + downloadFileName)
	expectedHash := ""

	// Try to find matching filename in checksums file
	for _, line := range strings.Split(fileContent, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		hash := strings.ToLower(fields[0])
		filename := fields[len(fields)-1]

		// Check if this line matches the download filename
		if filename == downloadFileName {
			logger.PrintIfVerbose("Found matching checksum entry for: " + filename)
			expectedHash = hash
			break
		}
	}

	// If no exact match found, fall back to first line (single-line format)
	if expectedHash == "" {
		fields := strings.Fields(fileContent)
		if len(fields) < 1 {
			return errors.New("Invalid checksum file format - no hash found")
		}
		expectedHash = strings.ToLower(fields[0])
	}

	if len(expectedHash) != 64 {
		return errors.Errorf("Invalid hash length - expected 64 hex characters, got %d", len(expectedHash))
	}

	actualHash, err := calculateSHA256(archivePath)
	if err != nil {
		return errors.Errorf("Failed to calculate archive hash: %s", err.Error())
	}

	logger.PrintIfVerbose(fmt.Sprintf("Actual Hash in Checksum.txt: %s", expectedHash))
	logger.PrintIfVerbose(fmt.Sprintf("Actual Hash of Zip: %s", actualHash))

	if !strings.EqualFold(expectedHash, actualHash) {
		return errors.New("Checksum verification failed - archive may have been tampered with")
	}

	logger.PrintIfVerbose("Archive Checksum Verification Successful.")
	return nil
}

// calculateSHA256 calculates the SHA256 hash of a file
func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
