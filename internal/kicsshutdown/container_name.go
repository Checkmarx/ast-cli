// Package kicsshutdown holds the current KICS Docker container name for SIGTERM cleanup.
// It is updated alongside viper wherever KicsContainerNameKey is set so the signal handler
// can read the latest name without concurrent access to viper.
package kicsshutdown

import "sync"

var (
	mu   sync.RWMutex
	name string
)

// SetKicsContainerName records the container name used for shutdown handling.
func SetKicsContainerName(n string) {
	mu.Lock()
	defer mu.Unlock()
	name = n
}

// GetKicsContainerName returns the last recorded container name.
func GetKicsContainerName() string {
	mu.RLock()
	defer mu.RUnlock()
	return name
}
