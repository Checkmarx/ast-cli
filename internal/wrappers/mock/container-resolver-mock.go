package mock

type ContainerResolverMockWrapper struct {
}

func (c *ContainerResolverMockWrapper) Resolve(scanPath string, resolutionFilePath string, images []string, isDebug bool) error {
	return nil
}
