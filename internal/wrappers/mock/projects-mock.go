package mock

import (
	"fmt"
)

type ProjectsMockWrapper struct{}

func (p *ProjectsMockWrapper) Create(model *Project) (
	*ProjectResponseModel,
	*ErrorModel,
	error) {
	fmt.Println("Called Create in ProjectsMockWrapper")
	return &ProjectResponseModel{
		Name: model.Name,
	}, nil, nil
}

func (p *ProjectsMockWrapper) Get(params map[string]string) (
	*ProjectsCollectionResponseModel,
	*ErrorModel,
	error) {
	fmt.Println("Called Get in ProjectsMockWrapper")

	filteredTotalCount := 1

	if params["name"] == "MOCK-NO-FILTERED-PROJECTS" {
		filteredTotalCount = 0
	}

	return &ProjectsCollectionResponseModel{
		FilteredTotalCount: uint(filteredTotalCount),
		Projects: []ProjectResponseModel{
			{
				ID:   "MOCK",
				Name: "MOCK",
			},
		},
	}, nil, nil
}

func (p *ProjectsMockWrapper) GetByID(projectID string) (
	*ProjectResponseModel,
	*ErrorModel,
	error) {
	fmt.Println("Called GetByID in ProjectsMockWrapper")
	return &ProjectResponseModel{
		ID: projectID,
		Tags: map[string]string{
			"a": "b",
			"c": "d",
		},
		Groups: []string{
			"a",
			"b",
		},
	}, nil, nil
}

func (p *ProjectsMockWrapper) GetBranchesByID(_ string, _ map[string]string) (
	[]string,
	*ErrorModel,
	error) {
	fmt.Println("Called GetBranchesByID in ProjectsMockWrapper")
	return []string{
		"master",
		"feature/MOCK",
	}, nil, nil
}

func (p *ProjectsMockWrapper) Delete(_ string) (
	*ErrorModel,
	error) {
	fmt.Println("Called Delete in ProjectsMockWrapper")
	return nil, nil
}

func (p *ProjectsMockWrapper) Tags() (
	map[string][]string,
	*ErrorModel,
	error) {
	fmt.Println("Called Tags in ProjectsMockWrapper")
	return map[string][]string{
		"t1": {
			"v1",
		},
	}, nil, nil
}
