package cleandata

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
)

const ProjectNameFile = "projectName.txt"

func DeleteProjectByName(projectName string) {
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(viper.GetString(params.ProjectsPathKey))
	projectModel, _, err := projectsWrapper.GetByName(projectName)
	if err == nil && projectModel != nil {
		_, _ = projectsWrapper.Delete(projectModel.ID)
	}
}

func TestDeleteProjectsFromFile(t *testing.T) {
	projectNameFile := fmt.Sprint("../integration/", ProjectNameFile) // Replace with your actual file path

	file, err := os.Open(projectNameFile)
	if err != nil {
		log.Printf("Failed to open project name file: %v", err)
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		projectName := scanner.Text()
		log.Printf("Attempting to delete project: %s", projectName)
		DeleteProjectByName(projectName)
		log.Printf("Project deleted: %s", projectName)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading project name file: %v", err)
	}
}
