package wrappers

type UploadsWrapper interface {
	Create(sourcesFile string) (*string, error)
}
