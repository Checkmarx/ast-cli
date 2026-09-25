package mock

import (
	"strings"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type SastMetadataMockWrapper struct{}

const (
	scanIDParam = "scan-ids"
	concurrent  = "ConcurrentTest"
	// FakeMetadataErrorID is used to trigger error responses in mock SAST metadata calls.
	FakeMetadataErrorID = "fake-sast-metadata-error-id"
	// FakeMetadataEmptyID is used to trigger empty responses in mock SAST metadata calls.
	FakeMetadataEmptyID = "fake-sast-metadata-empty-id"
)

func (s SastMetadataMockWrapper) GetSastMetadataByIDs(params map[string]string) (
	*wrappers.SastMetadataModel,
	error,
) {
	if strings.Contains(params[scanIDParam], FakeMetadataErrorID) {
		return nil, errors.New("fake sast metadata fetch error")
	}
	if strings.Contains(params[scanIDParam], FakeMetadataEmptyID) {
		return &wrappers.SastMetadataModel{TotalCount: 0, Scans: []wrappers.Scans{}}, nil
	}
	if strings.Contains(params[scanIDParam], concurrent) {
		scanIDs := strings.Split(params[scanIDParam], ",")
		return &wrappers.SastMetadataModel{
			TotalCount: len(scanIDs),
			Scans:      convertScanIDsToScans(scanIDs),
		}, nil
	}
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
				IncrementalCancelReason: "some reason",
				BaseID:                  "baseID",
				DeletedFilesCount:       3,
			},
		},
		Missing: []string{"missing1", "missing2"},
	}, nil
}

func convertScanIDsToScans(scanIDs []string) []wrappers.Scans {
	scans := make([]wrappers.Scans, 0, len(scanIDs))
	for i, scanID := range scanIDs {
		scans = append(scans, wrappers.Scans{
			ScanID:            scanID,
			ProjectID:         "project" + scanID,
			Loc:               100 + i,
			FileCount:         50 + i,
			IsIncremental:     i%2 == 0,
			AddedFilesCount:   10 + i,
			ChangedFilesCount: 5 + i,
		})
	}
	return scans
}
