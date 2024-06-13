package wrappers

type UploadsWrapper interface {
	UploadFile(sourcesFile string, featureFlagsWrapper FeatureFlagsWrapper) (*string, error)
}
