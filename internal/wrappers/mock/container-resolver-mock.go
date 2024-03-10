package mock

import (
	"io/ioutil"
	"path/filepath"
)

type ContainerResolverMockWrapper struct {
}

func (c *ContainerResolverMockWrapper) Resolve(scanPath, resolutionFilePath string, images []string, isDebug bool) error {
	// Create the content for the container-resolution.json file (empty JSON)
	content := []byte("{}\n")

	// Construct the full path for the container-resolution.json file
	resolutionFullPath := filepath.Join(scanPath, "containers-resolution.json")

	// Write the content to the container-resolution.json file
	err := ioutil.WriteFile(resolutionFullPath, content, 0644)
	if err != nil {
		return err
	}

	return nil
}
