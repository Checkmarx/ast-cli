package wrappers

type UploadsWrapper interface {
	UploadFile(sourcesFile string, featureFlagsWrapper FeatureFlagsWrapper) (*string, error)
	UploadFileInMultipart(path string, wrapper FeatureFlagsWrapper) (*string, error)
}
