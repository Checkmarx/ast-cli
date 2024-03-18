package wrappers

type ByorWrapper interface {
	Import(projectID, uploadURL string) (string, error)
}

type CreateImportsRequest struct {
	ProjectID string `json:"projectId"`
	UploadURL string `json:"UploadUrl"`
}

type CreateImportsResponse struct {
	ImportID string `json:"importId"`
}
