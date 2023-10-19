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
		if strings.Contains(str, v) {
			return true
		}
	}
	return false
}

func DefaultMapValue(params map[string]string, key, value string) {
	if _, ok := params[key]; !ok {
		params[key] = value
	}
}
