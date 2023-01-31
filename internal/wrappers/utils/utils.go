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

func ToStringArray(obj interface{}) []string {
	switch t := obj.(type) {
	case []interface{}:
		result := make([]string, 0, len(t))
		for _, v := range t {
			result = append(result, v.(string))
		}
		return result
	default:
		return []string{}
	}
}
