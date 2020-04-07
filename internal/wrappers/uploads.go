package wrappers

type UploadsWrapper interface {
	UploadFile(sourcesFile string) (*string, error)
}
