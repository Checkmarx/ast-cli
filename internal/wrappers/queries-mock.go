package wrappers

import (
	"io"
	"io/ioutil"
	"strings"
	"time"

	queries "github.com/checkmarxDev/sast-queries/pkg/v1/queriesobjects"
	queriesHelpers "github.com/checkmarxDev/sast-queries/pkg/web/helpers"
)

type QueriesMockWrapper struct{}

func (QueriesMockWrapper) Clone(string) (io.ReadCloser, *queriesHelpers.WebError, error) {
	return ioutil.NopCloser(strings.NewReader("mock")), nil, nil
}

func (QueriesMockWrapper) Import(string, string) (*queriesHelpers.WebError, error) {
	return nil, nil
}

func (QueriesMockWrapper) List() ([]*queries.QueriesRepo, error) {
	return []*queries.QueriesRepo{
		{Name: "mock", IsActive: true, LastModified: time.Now()},
	}, nil
}

func (QueriesMockWrapper) Activate(string) (*queriesHelpers.WebError, error) {
	return nil, nil
}

func (QueriesMockWrapper) Delete(string) (*queriesHelpers.WebError, error) {
	return nil, nil
}
