package wrappers

import (
	containersResolver "github.com/Checkmarx-Containers/containers-resolver/pkg/containerResolver"
)

type ContainerResolverWrapper interface {
	Resolve(scanPath string, resolutionFilePath string, images []string, isDebug bool) error
}

type ContainerResolverImpl struct {
}

func NewContainerResolverWrapper() ContainerResolverWrapper {
	return &ContainerResolverImpl{}
}

func (c *ContainerResolverImpl) Resolve(scanPath string, resolutionFilePath string, images []string, isDebug bool) error {
	return containersResolver.Resolve(scanPath, resolutionFilePath, images, isDebug)
}
