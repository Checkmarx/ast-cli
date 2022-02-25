package wrappers

type CodeBashingCollection struct {
	Path        string `json:"path,omitempty"`
	CweID       string `json:"cwe_id,omitempty"`
	Language    string `json:"lang,omitempty"`
	CxQueryName string `json:"cxQueryName,omitempty"`
}

type CodeBashingParamsCollection struct {
	CweID       string `json:"cwe_id,omitempty"`
	Language    string `json:"lang,omitempty"`
	CxQueryName string `json:"cxQueryName,omitempty"`
}
