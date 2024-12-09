package wrappers

import containersResolver "github.com/Checkmarx/containers-resolver/pkg/containerResolver"

type ContainerResolverWrapper interface {
	Resolve(scanPath string, resolutionFilePath string, images []string, isDebug bool) error
}

type ContainerResolverImpl struct {
	containersResolver.ContainersResolver
}

func NewContainerResolverWrapper() ContainerResolverWrapper {
	return &ContainerResolverImpl{
		containersResolver.NewContainerResolver(),
	}
}

func (c *ContainerResolverImpl) Resolve(scanPath, resolutionFilePath string, images []string, isDebug bool) error {
	return c.Resolve(scanPath, resolutionFilePath, images, isDebug)
}
