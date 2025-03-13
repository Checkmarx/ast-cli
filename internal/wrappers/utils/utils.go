package utils

import (
	"net/url"
	"path"
	"strings"
)

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
