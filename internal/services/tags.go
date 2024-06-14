package services

import "strings"

func createTagMap(tagListStr string) map[string]string {
	tagsList := strings.Split(tagListStr, ",")
	tags := make(map[string]string)
	for _, tag := range tagsList {
		if len(tag) > 0 {
			value := ""
			keyValuePair := strings.Split(tag, ":")
			if len(keyValuePair) > 1 {
				value = keyValuePair[1]
			}
			tags[keyValuePair[0]] = value
		}
	}
	return tags
}
