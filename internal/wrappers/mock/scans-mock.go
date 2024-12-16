package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var HasScs bool

type ScansMockWrapper struct {
	Running bool
}

func (m *ScansMockWrapper) GetWorkflowByID(_ string) ([]*wrappers.ScanTaskResponseModel, *wrappers.ErrorModel, error) {
	return nil, nil, nil
}

func (m *ScansMockWrapper) Create(scanModel *wrappers.Scan) (*wrappers.ScanResponseModel, *wrappers.ErrorModel, error) {
	fmt.Println("Called Create in ScansMockWrapper")
	if scanModel.Project.ID == "fake-kics-scanner-fail-id" {
		return &wrappers.ScanResponseModel{
			ID:     "fake-scan-id-kics-scanner-fail",
			Status: "MOCK",
		}, nil, nil
	}
	if scanModel.Project.ID == "fake-multiple-scanner-fails-id" {
		return &wrappers.ScanResponseModel{
			ID:     "fake-scan-id-multiple-scanner-fails",
			Status: "MOCK",
		}, nil, nil
	}
	if scanModel.Project.ID == "fake-sca-fail-partial-id" {
		return &wrappers.ScanResponseModel{
			ID:     "fake-scan-id-sca-fail-partial-id",
			Status: "MOCK",
		}, nil, nil
	}
	if scanModel.Project.ID == "fake-kics-fail-sast-canceled-id" {
		return &wrappers.ScanResponseModel{
			ID:     "fake-scan-id-kics-fail-sast-canceled-id",
			Status: "MOCK",
		}, nil, nil
	}

	return &wrappers.ScanResponseModel{
		ID:     uuid.New().String(),
		Status: "MOCK",
	}, nil, nil
}

func (m *ScansMockWrapper) Get(scanParams map[string]string) (
	*wrappers.ScansCollectionResponseModel,
	*wrappers.ErrorModel,
	error,
) {
	if scanParams["project-id"] == "non-existent-project" {
		return &wrappers.ScansCollectionResponseModel{}, nil, nil
	}

	fmt.Println("Called Get in ScansMockWrapper")
	sastMapConfig := make(map[string]interface{})
	sastMapConfig["incremental"] = "trueSastIncremental"
	sastMapConfig["presetName"] = "preset"
	sastMapConfig["filter"] = "filterValueSast"
	sastMapConfig["engineVerbose"] = "true"
	sastMapConfig["languageMode"] = "languageModeValue"
	var sastConfig = wrappers.Config{
		Type:  "sast",
		Value: sastMapConfig,
	}
	var configs []wrappers.Config
	configs = append(configs, sastConfig)

	kicsMapConfig := make(map[string]interface{})
	kicsMapConfig["platforms"] = "platformsValue"
	kicsMapConfig["filter"] = "filterValueKics"
	var kicsConfig = wrappers.Config{
		Type:  "kics",
		Value: kicsMapConfig,
	}
	configs = append(configs, kicsConfig)

	scaMapConfig := make(map[string]interface{})
	scaMapConfig["filter"] = "filterValueSca"
	var scaConfig = wrappers.Config{
		Type:  "sca",
		Value: scaMapConfig,
	}
	configs = append(configs, scaConfig)

	var metadata = wrappers.ScanResponseModelMetadata{Configs: configs}
	var engines []string
	engines = append(engines, "sast")
	return &wrappers.ScansCollectionResponseModel{
		Scans: []wrappers.ScanResponseModel{
			{
				ID:       "MOCK",
				Status:   "STATUS",
				Metadata: metadata,
				Engines:  engines,
			},
		},
	}, nil, nil
}

func (m *ScansMockWrapper) GetByID(scanID string) (*wrappers.ScanResponseModel, *wrappers.ErrorModel, error) {
	fmt.Println("Called GetByID in ScansMockWrapper")
	if scanID == "fake-error-id" {
		return nil, nil, errors.New("fake error message")
	}

	if scanID == "fake-scan-id-kics-scanner-fail" {
		return &wrappers.ScanResponseModel{
			ID:     "fake-scan-id-kics-scanner-fail",
			Status: wrappers.ScanFailed,
			StatusDetails: []wrappers.StatusInfo{
				{
					Status:    wrappers.ScanFailed,
					Name:      "kics",
					Details:   "error message from kics scanner",
					ErrorCode: 1234,
				},
			},
		}, nil, nil
	}
	if scanID == "fake-scan-id-multiple-scanner-fails" {
		return &wrappers.ScanResponseModel{
			ID:     "fake-scan-id-multiple-scanner-fails",
			Status: wrappers.ScanFailed,
			StatusDetails: []wrappers.StatusInfo{
				{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 2344},
				{Status: wrappers.ScanFailed, Name: "sca", Details: "error message from sca scanner", ErrorCode: 4343},
			},
		}, nil, nil
	}
	if scanID == "fake-scan-id-sca-fail-partial-id" {
		return &wrappers.ScanResponseModel{
			ID:     "fake-scan-id-sca-fail-partial-id",
			Status: wrappers.ScanPartial,
			StatusDetails: []wrappers.StatusInfo{
				{Status: wrappers.ScanCompleted, Name: "sast"},
				{Status: wrappers.ScanFailed, Name: "sca", Details: "error message from sca scanner", ErrorCode: 4343},
			},
		}, nil, nil
	}
	if scanID == "fake-scan-id-kics-fail-sast-canceled-id" {
		return &wrappers.ScanResponseModel{
			ID:     "fake-scan-id-kics-fail-sast-canceled-id",
			Status: wrappers.ScanCanceled,
			StatusDetails: []wrappers.StatusInfo{
				{Status: wrappers.ScanCompleted, Name: "general"},
				{Status: wrappers.ScanCompleted, Name: "sast"},
				{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 6455},
			},
		}, nil, nil
	}

	if scanID == "ScanRunning" {
		return &wrappers.ScanResponseModel{
			ID:        "ScanRunning",
			ProjectID: "Project1",
			Status:    "Running",
		}, nil, nil
	}
	if scanID == "ScanNotRunning" {
		return &wrappers.ScanResponseModel{
			ID:        "ScanNotRunning",
			ProjectID: "Project2",
			Status:    "Completed",
		}, nil, nil
	}

	var status wrappers.ScanStatus = "Completed"
	m.Running = !m.Running
	engines := []string{params.ScaType, params.SastType, params.KicsType, params.ContainersType}
	if HasScs {
		engines = append(engines, params.ScsType)
	}
	return &wrappers.ScanResponseModel{
		ID:      scanID,
		Status:  status,
		Engines: engines,
	}, nil, nil
}

func (m *ScansMockWrapper) Delete(_ string) (*wrappers.ErrorModel, error) {
	fmt.Println("Called Delete in ScansMockWrapper")
	return nil, nil
}

func (m *ScansMockWrapper) Cancel(string) (*wrappers.ErrorModel, error) {
	fmt.Println("Called Cancel in ScansMockWrapper")
	return nil, nil
}

func (m *ScansMockWrapper) Tags() (map[string][]string, *wrappers.ErrorModel, error) {
	fmt.Println("Called Tags in ScansMockWrapper")
	return map[string][]string{"t1": {"v1"}}, nil, nil
}
