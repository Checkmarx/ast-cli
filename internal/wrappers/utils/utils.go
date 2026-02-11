package utils

import (
	"errors"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/checkmarx/ast-cli/internal/logger"
)

var (
	mutex          sync.RWMutex
	optionalParams = make(map[string]string)
)

var allowedOptionalKeys = map[string]bool{
	"asca-location": true,
}

// CleanURL returns a cleaned url removing double slashes
func CleanURL(uri string) (string, error) {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	parsedURL.Path = path.Clean(parsedURL.Path)
	return parsedURL.String(), err
}

func Contains(s []string, str string) bool {
	for _, v := range s {
		if strings.Contains(v, str) {
			return true
		}
	}
	return false
}

func LimitSlice[T any](slice []T, limit int) []T {
	if limit <= 0 || limit >= len(slice) {
		return slice
	}
	return slice[:limit]
}

func SetOptionalParam(key, value string) error {
	logger.PrintfIfVerbose("Setting optional parameter: %s", key)
	mutex.Lock()
	defer mutex.Unlock()
	value = strings.TrimSpace(value)
	key = strings.TrimSpace(key)
	if _, ok := allowedOptionalKeys[key]; ok {
		optionalParams[key] = value
	} else {
		return errors.New("Invalid optional parameter key: " + key)
	}
	return nil
}

func hasOptionalParam(key string) bool {
	mutex.RLock()
	defer mutex.RUnlock()
	_, ok := optionalParams[key]
	return ok
}

func GetOptionalParam(key string) string {
	mutex.RLock()
	defer mutex.RUnlock()
	trimmedKey := strings.TrimSpace(key)
	if hasOptionalParam(trimmedKey) {
		return optionalParams[trimmedKey]
	}
	return ""
}
