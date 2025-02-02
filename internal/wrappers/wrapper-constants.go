package wrappers

import "time"

const (
	limitValue    = "10000"
	retryAttempts = 4
	retryDelay    = 500 * time.Millisecond
)
