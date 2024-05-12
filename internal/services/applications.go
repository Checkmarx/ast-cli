package services

import "github.com/checkmarx/ast-cli/internal/wrappers/utils"

func createApplicationIds(applicationID, existingApplicationIds []string) []string {
	for _, id := range applicationID {
		if !utils.Contains(existingApplicationIds, id) {
			existingApplicationIds = append(existingApplicationIds, id)
		}
	}
	return existingApplicationIds
}
