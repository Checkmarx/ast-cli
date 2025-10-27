package wrappers

type CodeBashingTag struct {
	UUID  string `json:"uuid,omitempty"`
	CWE   string `json:"cwe,omitempty"`
	OWASP string `json:"owasp,omitempty"`
	SANS  string `json:"sans,omitempty"`
}

type CodeBashingCollection struct {
	LessonDisplayName string           `json:"lessonDisplayName,omitempty"`
	CourseDisplayName string           `json:"courseDisplayName,omitempty"`
	Duration          string           `json:"duration,omitempty"`
	ImageUrl          string           `json:"imageUrl,omitempty"`
	TagsResult        []CodeBashingTag `json:"tagsResult,omitempty"`
	Path              string           `json:"lessonUrl,omitempty"`
	PotentialPoints   int              `json:"potentialPoints,omitempty"`
}

// Wrapper struct to handle API responses with "data" field
type CodeBashingResponse struct {
	Data CodeBashingCollection `json:"data,omitempty"`
}

type CodeBashingParamsCollection struct {
	QueryId string `json:"queryId,omitempty"`
}
