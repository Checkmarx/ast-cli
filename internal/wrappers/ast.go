package wrappers

import (
	"encoding/json"
	"fmt"
)

type ErrorModel struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

type WebError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type AstError struct {
	Code int
	Err  error
}

func (er *AstError) Error() string {
	return fmt.Sprintf("%v", er.Err)
}

func (er *AstError) Unwrap() error {
	return er.Err
}

func NewAstError(code int, err error) *AstError {
	return &AstError{
		Code: code,
		Err:  err,
	}
}
