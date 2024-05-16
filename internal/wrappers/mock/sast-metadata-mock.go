package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type SastMetadataMockWrapper struct{}

func (s SastMetadataMockWrapper) GetSastMetadataByIDs(scanIds []string) (
	*wrappers.SastMetadataModel,
	error,
) {
	return &wrappers.SastMetadataModel{
		TotalCount: 2,
		Scans: []wrappers.Scans{
			{
				ScanID:            "scan1",
				ProjectID:         "project1",
				Loc:               100,
				FileCount:         50,
				IsIncremental:     true,
				AddedFilesCount:   10,
				ChangedFilesCount: 5,
			},
			{
				ScanID:                  "scan2",
				ProjectID:               "project2",
				Loc:                     150,
				FileCount:               70,
				IsIncremental:           false,
				IsIncrementalCanceled:   true,
				IncrementalCancelReason: "Some reason",
				BaseID:                  "baseID",
				DeletedFilesCount:       3,
			},
		},
		Missing: []string{"missing1", "missing2"},
	}, nil
}
