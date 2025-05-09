package wrappers

import (
	"strings"
)

type EngineHTTPWrapper struct {
}

func (e *EngineHTTPWrapper) Get(engineName string) *EngineListResponseModel {
	return e.getEngineList(engineName)
}

func NewHttpEnginesWrapper() *EngineHTTPWrapper {
	return &EngineHTTPWrapper{}
}

func (e *EngineHTTPWrapper) getEngineList(engineName string) *EngineListResponseModel {
	allEngines := &EngineListResponseModel{
		Engines: []EngineList{
			{
				EngineId:   "1",
				EngineName: "SAST",
				APIs: []EngineAPIs{
					{
						ApiURL:      "https://{HostName}/api/v1/scans",
						ApiName:     "Get -> SAST Current Scans",
						Description: "Gets List of current Scans",
					},
					{
						ApiURL:      "https://{HostName}/api/v1/scans/{id}/status",
						ApiName:     "Get -> SAST Scans status",
						Description: "Retrieve that current status of Scan",
					},
					{
						ApiURL:      "https://{HostName}/api/v1/scans/{id}/results",
						ApiName:     "Get -> SAST Scans results",
						Description: "Retrieve scan results",
					},
				},
			},
			{
				EngineId:   "2",
				EngineName: "SCA",
				APIs: []EngineAPIs{
					{
						ApiURL:      "https://{HOST_NAME}/api/scans/{scanId}",
						ApiName:     "Get -> SCA scan details",
						Description: "Retriever SCA scan details and status",
					},
					{
						ApiURL:      "https://{HOST_NAME}/api/scans",
						ApiName:     "Post -> Create a new SCA scan",
						Description: "Create new scan and get the vulnerabilities in a packages",
					},
				},
			},
		},
	}

	if engineName != "" {
		var filteredEngines []EngineList
		for _, engine := range allEngines.Engines {
			if strings.ToLower(engine.EngineName) == strings.ToLower(engineName) {
				filteredEngines = append(filteredEngines, engine)
			}
		}

		return &EngineListResponseModel{
			Engines: filteredEngines,
		}
	}
	return allEngines
}
