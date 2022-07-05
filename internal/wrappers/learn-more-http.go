package wrappers

type LearnMoreHTTPWrapper struct {
	path string
}

func newHTTPLearnMoreWrapper(path string) *LearnMoreHTTPWrapper {
	return &LearnMoreHTTPWrapper{
		path: path,
	}
}

func (r *LearnMoreHTTPWrapper) GetLearnMoreDetails(queryId string) (
	*LearnMoreResponseModel,
	*WebError,
	error,
) {
	return nil, nil, nil
}
