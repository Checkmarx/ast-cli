package wrappers

import (
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/pkg/errors"
	"strings"
)

type EngineHTTPWrapper struct {
	path        string
	contentType string
}

//get engine API

func (e *EngineHTTPWrapper) Get(engineName string) (*EnginesCollectionResponseModel, error) {

	return e.getEnginesApiResponse(engineName)
}

//mocking the response of enginesApiResponse

func (e *EngineHTTPWrapper) getEnginesApiResponse(engineName string) (*EnginesCollectionResponseModel, error) {

	if strings.ToLower(engineName) == "sast" {
		return &EnginesCollectionResponseModel{
			Engines: []EngineResponseModel{
				{
					EngineID:   "1",
					EngineName: "SAST",
					Apis: []EngineAPIModel{
						{
							ApiURL:      "https://{HOST_NAME}/api/sast-results",
							ApiName:     "SAST Results API",
							Description: "Gets SAST Results for Scans",
						},
						{
							ApiURL:      "https://{HOST_NAME}/api/sast-metadata",
							ApiName:     "SAST Meta-data API ",
							Description: "Gets SAST Metadata for one or more scans",
						},
					},
				},
			},
		}, nil
	}

	if strings.ToLower(engineName) == "sca" {
		return &EnginesCollectionResponseModel{
			Engines: []EngineResponseModel{
				{
					EngineID:   "2",
					EngineName: "SCA",
					Apis: []EngineAPIModel{
						{
							ApiURL:      "https://{HOST_NAME}/api/sca/analysis/requests",
							ApiName:     "SCA File Analysis  API",
							Description: "Gets Checkmarx SCA File Analysis Service REST API",
						},
						{
							ApiURL:      "https://{HOST_NAME}/api/sca/management-of-risk",
							ApiName:     "SCA Scanner- Management of Risk REST API ",
							Description: "Crud operations with SCA Scanner-Management of Risk REST API",
						},
					},
				},
			},
		}, nil
	}

	if strings.ToLower(engineName) == "dast" {
		return &EnginesCollectionResponseModel{
			Engines: []EngineResponseModel{
				{
					EngineID:   "3",
					EngineName: "DAST",
					Apis: []EngineAPIModel{
						{
							ApiURL:      "https://{HOST_NAME}/api/dast/scans",
							ApiName:     "SCA File Analysis  API",
							Description: "Gets Checkmarx SCA File Analysis Service REST API",
						},
						{
							ApiURL:      "https://{HOST_NAME}/api/dast/mfe-results",
							ApiName:     "DAST Results Service REST API ",
							Description: "API to interact with Gets the DAST Results Service",
						},
					},
				},
			},
		}, nil
	}

	if engineName != "" && (strings.ToLower(engineName) != "sast" || strings.ToLower(engineName) != "sca" || strings.ToLower(engineName) != "dast") {
		return nil, errors.Errorf(errorConstants.EngineDoesNotExist)
	}

	return &EnginesCollectionResponseModel{
		Engines: []EngineResponseModel{
			{
				EngineID:   "1",
				EngineName: "SAST",
				Apis: []EngineAPIModel{
					{
						ApiURL:      "https://{HOST_NAME}/api/sast-results",
						ApiName:     "SAST Results API",
						Description: "Gets SAST Results for Scans",
					},
					{
						ApiURL:      "https://{HOST_NAME}/api/sast-metadata",
						ApiName:     "SAST Meta-data API ",
						Description: "Gets SAST Metadata for one or more scans",
					},
				},
			},
			{
				EngineID:   "2",
				EngineName: "SCA",
				Apis: []EngineAPIModel{
					{
						ApiURL:      "https://{HOST_NAME}/api/sca/analysis/requests",
						ApiName:     "SCA File Analysis  API",
						Description: "Gets Checkmarx SCA File Analysis Service REST API",
					},
					{
						ApiURL:      "https://{HOST_NAME}/api/sca/management-of-risk",
						ApiName:     "SCA Scanner- Management of Risk REST API ",
						Description: "Crud operations with SCA Scanner-Management of Risk REST API",
					},
				},
			},
			{
				EngineID:   "3",
				EngineName: "DAST",
				Apis: []EngineAPIModel{
					{
						ApiURL:      "https://{HOST_NAME}/api/dast/scans",
						ApiName:     "DAST Scans Service REST API",
						Description: "Gets Checkmarx DAST Scans Service REST API",
					},
					{
						ApiURL:      "https://{HOST_NAME}/api/dast/mfe-results",
						ApiName:     "DAST Results Service REST API ",
						Description: "API to interact with Gets the DAST Results Service",
					},
				},
			},
		},
	}, nil
}

// Constructor function to initialize

func NewHTTPEngineWrapper(path string) *EngineHTTPWrapper {
	return &EngineHTTPWrapper{
		path:        path,
		contentType: "application/json",
	}
}
