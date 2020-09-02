package wrappers

import (
	"io"

	queries "github.com/checkmarxDev/sast-queries/pkg/v1/queriesobjects"
	queriesHelpers "github.com/checkmarxDev/sast-queries/pkg/web/helpers"
)

type QueriesWrapper interface {
	Clone(name string) (io.ReadCloser, *queriesHelpers.WebError, error)
	Import(uploadURL string, name string) (*queriesHelpers.WebError, error)
	List() ([]*queries.QueriesRepo, error)
	Activate(name string) (*queriesHelpers.WebError, error)
	Delete(name string) (*queriesHelpers.WebError, error)
}
