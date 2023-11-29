package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type SastMetadataMockWrapper struct {
}

func (s *SastMetadataMockWrapper) GetSastMetadataByIDs(params map[string]string) (*wrappers.SastMetadataModel, error) {
	return &wrappers.SastMetadataModel{
		TotalCount: 2,
		Scans: []wrappers.Scans{
			{
				ScanId:            "scan1",
				ProjectId:         "project1",
				Loc:               100,
				FileCount:         50,
				IsIncremental:     true,
				AddedFilesCount:   10,
				ChangedFilesCount: 5,
			},
			{
				ScanId:                  "scan2",
				ProjectId:               "project2",
				Loc:                     150,
				FileCount:               70,
				IsIncremental:           false,
				IsIncrementalCanceled:   true,
				IncrementalCancelReason: "Some reason",
				BaseId:                  "baseID",
				DeletedFilesCount:       3,
			},
		},
		Missing: []string{"missing1", "missing2"},
	}, nil
}
