package mock

import "fmt"

type UploadsMockWrapper struct {
}

func (u *UploadsMockWrapper) UploadFile(_ string) (*string, error) {
	fmt.Println("Called Create in UploadsMockWrapper")
	url := "/path/to/nowhere"
	return &url, nil
}
