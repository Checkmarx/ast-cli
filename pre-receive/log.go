package pre_receive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// logReport writes the given report to a file named by creation time.
func logReport(folderPath, scanReport string) error {
	if folderPath == "" {
		return nil
	}

	info, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log folder %q does not exist", folderPath)
		}
		return fmt.Errorf("unable to stat folder %q: %w", folderPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q exists but is not a directory", folderPath)
	}

	now := time.Now().UTC()
	timestamp := now.Format("2006-01-02_15-04-05.000000000")
	fileName := fmt.Sprintf("report_%s.log", timestamp)

	logFilePath := filepath.Join(folderPath, fileName)

	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %q: %w", logFilePath, err)
	}
	defer f.Close()

	if _, err = f.WriteString(scanReport); err != nil {
		return fmt.Errorf("failed to write to log file %q: %w", logFilePath, err)
	}

	return nil
}

// logSkip writes a one-off skip log named skip_<timestamp>.log,
// including exactly which refs were pushed.
func logSkip(folderPath string, refs []string) error {
	info, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log folder %q does not exist", folderPath)
		}
		return fmt.Errorf("unable to stat folder %q: %w", folderPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q exists but is not a directory", folderPath)
	}

	ts := time.Now().UTC().Format("2006-01-02_15-04-05.000000000")

	filePath := filepath.Join(folderPath, fmt.Sprintf("skip_%s.log", ts))

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open skip log file %q: %w", filePath, err)
	}
	defer f.Close()

	header := "Push skipped by secret scanner for refs:\n"
	if _, err = f.WriteString(header); err != nil {
		return fmt.Errorf("writing header to skip log: %w", err)
	}
	for _, r := range refs {
		if _, err = f.WriteString(r + "\n"); err != nil {
			return fmt.Errorf("writing ref to skip log: %w", err)
		}
	}
	return nil
}
