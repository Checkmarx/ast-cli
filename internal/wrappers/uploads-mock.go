package wrappers

import "fmt"

type UploadsMockWrapper struct {
}

func (u *UploadsMockWrapper) UploadFile(sourcesFile string) (*string, error) {
	fmt.Println("Called Create in UploadsMockWrapper")
	url := "/pathFormat/to/nowhere"
	return &url, nil
}
