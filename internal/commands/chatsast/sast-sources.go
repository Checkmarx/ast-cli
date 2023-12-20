package chatsast

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func GetSourcesForResult(scanResult Result, sourceDir string) (map[string][]string, error) {
	sourceFilenames := make(map[string]bool)
	for _, node := range scanResult.Data.Nodes {
		sourceFilename := strings.ReplaceAll(node.FileName, "\\", "/")
		sourceFilenames[sourceFilename] = true
	}

	fileContents, err := GetFileContents(sourceFilenames, sourceDir)
	if err != nil {
		return nil, err
	}

	return fileContents, nil
}

func GetSourcesForQuery(scanResults *ScanResults, sourceDir, language, query string) (map[string][]string, error) {
	sourceFilenames := make(map[string]bool)
	for _, scanResult := range scanResults.Results {
		if scanResult.Data.LanguageName != language || scanResult.Data.QueryName != query {
			continue
		}
		for _, node := range scanResult.Data.Nodes {
			sourceFilename := strings.ReplaceAll(node.FileName, "\\", "/")
			sourceFilenames[sourceFilename] = true
		}
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
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		fileContents[filename] = lines
	}

	return fileContents, nil
}
