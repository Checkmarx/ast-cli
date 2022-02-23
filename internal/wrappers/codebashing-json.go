package wrappers

type CodeBashingCollection struct {
	Path        string `json:"path,omitempty"`
	CweID       string `json:"cwe_id,omitempty"`
	Lang        string `json:"lang,omitempty"`
	CxQueryName string `json:"cxQueryName,omitempty"`
}
