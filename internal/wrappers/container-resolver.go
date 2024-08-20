package wrappers

type ContainerResolverWrapper interface {
	Resolve(scanPath string, resolutionFilePath string, images []string, isDebug bool) error
}

type ContainerResolverImpl struct {
}

func NewContainerResolverWrapper() ContainerResolverWrapper {
	return &ContainerResolverImpl{}
}

func (c *ContainerResolverImpl) Resolve(scanPath, resolutionFilePath string, images []string, isDebug bool) error {
	return nil
}
