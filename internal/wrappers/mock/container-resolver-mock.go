package mock

import (
	"os"
	"path/filepath"
)

type ContainerResolverMockWrapper struct {
}

func (c *ContainerResolverMockWrapper) Resolve(scanPath, resolutionFilePath string, images []string, isDebug bool) error {
	// Create the content for the container-resolution.json file (empty JSON)
	content := []byte("{}\n")

	resolutionFullPath := filepath.Join(scanPath, "containers-resolution.json")

	file, err := os.Create(resolutionFullPath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	_, err = file.Write(content)
	if err != nil {
		return err
	}

	return nil
}
