package scarealtime

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

var ScaResolverWorkingDir = filepath.Join(os.TempDir(), "SCARealtime")

var GetPackageManagerFromResolvingModuleType = map[string]string{
	"composer":  "Php",
	"gomodules": "Go",
	"pip":       "Python",
	"poetry":    "Python",
	"rubygems":  "Ruby",
	"npm":       "Npm",
	"yarn":      "Npm",
	"bower":     "Npm",
	"lerna":     "Npm",
	"sbt":       "Maven",
	"ivy":       "Maven",
	"maven":     "Maven",
	"gradle":    "Maven",
	"swiftpm":   "Ios",
	"carthage":  "Ios",
	"cocoapods": "Ios",
	"nuget":     "Nuget",
	"cpp":       "Cpp",
}

// downloadSCAResolverAndHashFileIfNeeded Downloads SCA Realtime if it is not downloaded yet
func downloadSCAResolverAndHashFileIfNeeded(scaRealTime *ScaRealTime) error {
	logger.PrintIfVerbose("Handling SCA Resolver...")

	if downloadNotNeeded(scaRealTime) {
		logger.PrintIfVerbose("SCA Resolver already exists and is up to date. Skipping download.")
		return nil
	}

	// Create temporary working directory if not exists
	err := createWorkingDirectory()
	if err != nil {
		return err
	}

	// Download SCA Resolver
	err = downloadFile(scaRealTime.SCAResolverDownloadURL, scaRealTime.SCAResolverFileName)
	if err != nil {
		return err
	}

	// Download SCA Resolver hash file
	err = downloadSCAResolverHashFile(scaRealTime.SCAResolverHashDownloadURL, scaRealTime.SCAResolverHashFileName)
	if err != nil {
		return err
	}

	// Unzip or extract SCA Resolver depending on which OS it is running
	err = UnzipOrExtractFiles()
	if err != nil {
		return err
	}

	return nil
}

// createWorkingDirectory Creates a working directory to handle SCA Realtime functionality
func createWorkingDirectory() error {
	logger.PrintIfVerbose("Creating temporary directory to handle SCA Realtime...")
	err := os.MkdirAll(ScaResolverWorkingDir, fs.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// downloadNotNeeded Checks if SCA Resolver is already available and if there is no need to download a new version
func downloadNotNeeded(scaRealTime *ScaRealTime) bool {
	logger.PrintIfVerbose("Checking if SCA Resolver already exists...")
	executableFileExists, _ := fileExists(scaRealTime.ExecutableFilePath)

	if !executableFileExists {
		return false
	}

	logger.PrintIfVerbose("SCA Resolver exists. Checking if it is the latest version...")

	isSCALastVersion, _ := isLastSCAResolverVersion(scaRealTime.HashFilePath, scaRealTime.SCAResolverHashDownloadURL, scaRealTime.SCAResolverHashFileName)

	return isSCALastVersion
}

// isLastSCAResolverVersion Checks if SCA Resolver is updated by comparing hashes
func isLastSCAResolverVersion(scaResolverHashFilePath, scaResolverHashURL, scaResolverZipFileNameHash string) (bool, error) {
	existingHash, _ := getHashValue(scaResolverHashFilePath)

	// Download SCA Resolver hash file
	err := downloadSCAResolverHashFile(scaResolverHashURL, scaResolverZipFileNameHash)
	if err != nil {
		return false, err
	}

	currentHash, _ := getHashValue(scaResolverHashFilePath)

	if !bytes.Equal(existingHash, currentHash) {
		logger.PrintIfVerbose("SCA Resolver is out of date.")
	}

	return bytes.Equal(existingHash, currentHash), nil
}

// fileExists Check if a file exists in a specific directory
func fileExists(path string) (bool, error) {
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
		log.Fatal(err)
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

// downloadSCAResolverHashFile Downloads SCA Resolver hash file
func downloadSCAResolverHashFile(scaResolverHashURL, scaResolverZipFileNameHash string) error {
	err := downloadFile(scaResolverHashURL, scaResolverZipFileNameHash)
	if err != nil {
		return err
	}

	return nil
}

// downloadFile Downloads a file
func downloadFile(downloadURLPath, fileName string) error {
	logger.PrintIfVerbose("Downloading " + fileName + " from: " + downloadURLPath)

	response, err := wrappers.SendHTTPRequestByFullURL(http.MethodGet, downloadURLPath, http.NoBody, false, 0, "", true)
	if err != nil {
		return errors.Errorf("Invoking HTTP request to upload file failed - %s", err.Error())
	}
	defer func() {
		_ = response.Body.Close()
	}()

	scaResolverZipFile, err := os.Create(filepath.Join(ScaResolverWorkingDir, fileName))
	if err != nil {
		fmt.Printf("Error creating SCA resolver zip file: %s", err)
		return err
	}
	defer func() {
		_ = scaResolverZipFile.Close()
	}()

	// Write the body to file
	_, err = io.Copy(scaResolverZipFile, response.Body)
	if err != nil {
		fmt.Printf("Error writing the body response to zip file: %s", err)
		return err
	}

	return nil
}
