package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"

	"github.com/google/uuid"
)

type ScansMockWrapper struct {
	Running bool
	HasSCS  bool
}

func (m *ScansMockWrapper) GetWorkflowByID(_ string) ([]*wrappers.ScanTaskResponseModel, *wrappers.ErrorModel, error) {
	return nil, nil, nil
}

func (m *ScansMockWrapper) Create(_ *wrappers.Scan) (*wrappers.ScanResponseModel, *wrappers.ErrorModel, error) {
	fmt.Println("Called Create in ScansMockWrapper")
	return &wrappers.ScanResponseModel{
		ID:     uuid.New().String(),
		Status: "MOCK",
	}, nil, nil
}

func (m *ScansMockWrapper) Get(_ map[string]string) (
	*wrappers.ScansCollectionResponseModel,
	*wrappers.ErrorModel,
	error,
) {
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
	var status wrappers.ScanStatus = "Completed"
	m.Running = !m.Running
	engines := []string{params.ScaType, params.SastType, params.KicsType}
	if m.HasSCS {
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
