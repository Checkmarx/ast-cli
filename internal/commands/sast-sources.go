package commands

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func GetSourcesForResult(scanResult *Result, sourceDir string) (map[string][]string, error) {
	sourceFilenames := make(map[string]bool)
	for i := range scanResult.Data.Nodes {
		sourceFilename := strings.ReplaceAll(scanResult.Data.Nodes[i].FileName, "\\", "/")
		sourceFilenames[sourceFilename] = true
	}

	fileContents, err := GetFileContents(sourceFilenames, sourceDir)
	if err != nil {
		return nil, err
	}

	return fileContents, nil
}

func GetFileContents(filenames map[string]bool, sourceDir string) (map[string][]string, error) {
	fileContents := make(map[string][]string)

	for filename := range filenames {
		sourceFilename := filepath.Join(sourceDir, filename)
		file, err := os.Open(sourceFilename)
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		err = file.Close()
		if err != nil {
			return nil, err
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		fileContents[filename] = lines
	}

	return fileContents, nil
}
