package wrappers

type CodeBashingCollection struct {
	LessonDisplayName string            `json:"lessonDisplayName,omitempty"`
	CourseDisplayName string            `json:"courseDisplayName,omitempty"`
	Duration          string            `json:"duration,omitempty"`
	ImageUrl          string            `json:"imageUrl,omitempty"`
	TagsResult        map[string]string `json:"tagsResult,omitempty"`
	Path              string            `json:"lessonUrl,omitempty"`
	PotentialPoints   string            `json:"potentialPoints,omitempty"`
}

// Wrapper struct to handle API responses with "data" field
type CodeBashingResponse struct {
	Data CodeBashingCollection `json:"data"`
}

type CodeBashingParamsCollection struct {
	QueryId string `json:"queryId,omitempty"`
}
