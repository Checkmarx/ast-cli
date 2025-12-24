package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type RisksOverviewHTTPWrapper struct {
	risksOverviewPath     string
	apiSecurityResultPath string
}

func NewHTTPRisksOverviewWrapper(risksOverviewPath, apiSecurityResultPath string) RisksOverviewWrapper {
	return &RisksOverviewHTTPWrapper{
		risksOverviewPath:     risksOverviewPath,
		apiSecurityResultPath: apiSecurityResultPath,
	}
}

type APISecRiskEntry struct {
	RiskID        string `json:"risk_id,omitempty"`
	APIID         string `json:"api_id,omitempty"`
	Severity      string `json:"severity,omitempty"`
	Name          string `json:"name,omitempty"`
	Status        string `json:"status,omitempty"`
	HTTPMethod    string `json:"http_method,omitempty"`
	URL           string `json:"url,omitempty"`
	Origin        string `json:"origin,omitempty"`
	Documented    bool   `json:"documented,omitempty"`
	Authenticated *bool  `json:"authenticated,omitempty"`
	DiscoveryDate string `json:"discovery_date,omitempty"`
	ScanID        string `json:"scan_id,omitempty"`
	SastRiskID    string `json:"sast_risk_id,omitempty"`
	ProjectID     string `json:"project_id,omitempty"`
	State         string `json:"state,omitempty"`
}

type APISecPaginatedResult struct {
	Entries            []APISecRiskEntry `json:"entries,omitempty"`
	TotalRecords       string            `json:"total_records,omitempty"`
	TotalPages         string            `json:"total_pages,omitempty"`
	HasPrevious        bool              `json:"has_previous,omitempty"`
	HasNext            bool              `json:"has_next,omitempty"`
	NextPageNumber     int               `json:"next_page_number,omitempty"`
	PreviousPageNumber *int              `json:"previous_page_number,omitempty"`
}

type APISecRiskEntriesResult struct {
	Entries []APISecRiskEntry
}

type FilterParam struct {
	Column   string   `json:"column"`
	Values   []string `json:"values"`
	Operator string   `json:"operator"`
}

func (r *RisksOverviewHTTPWrapper) GetAllAPISecRisksByScanID(scanID string) (
	*APISecResult,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := fmt.Sprintf(r.risksOverviewPath, scanID)
	resp, err := SendHTTPRequest(http.MethodGet, path, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := APISecResult{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func buildFiltersFromQueryParam(queryParam map[string]string) []FilterParam {
	var filters []FilterParam
	allowed := map[string]struct{}{"state": {}, "status": {}, "severity": {}}
	for k, v := range queryParam {
		if _, ok := allowed[k]; !ok {
			continue
		}
		var values []string
		if v != "" {
			for _, val := range strings.Split(v, ",") {
				values = append(values, strings.ToLower(strings.TrimSpace(val)))
			}
		}
		filters = append(filters, FilterParam{
			Column:   k,
			Values:   values,
			Operator: "in",
		})
	}
	return filters
}

func (r *RisksOverviewHTTPWrapper) GetFilterResultForAPISecByScanID(scanID string, queryParam map[string]string) (APISecRiskEntriesResult, *WebError, error) {
	allEntries := make([]APISecRiskEntry, 0)
	page := 1
	filters := buildFiltersFromQueryParam(queryParam)
	for {
		queryParam["page"] = fmt.Sprintf("%d", page)
		result, webErr, err := getAPISecPaginatedResultPage(r.apiSecurityResultPath, scanID, queryParam, filters)
		if err != nil || webErr != nil {
			return APISecRiskEntriesResult{Entries: allEntries}, webErr, err
		}
		allEntries = append(allEntries, result.Entries...)
		if !result.HasNext || result.NextPageNumber == 0 || len(result.Entries) == 0 {
			break
		}
		page = result.NextPageNumber
	}
	delete(queryParam, "page")
	return APISecRiskEntriesResult{Entries: allEntries}, nil, nil
}

func getAPISecPaginatedResultPage(apiSecurityResultPath string, scanID string, queryParam map[string]string, filters []FilterParam) (*APISecPaginatedResult, *WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	var filteringJSON []byte
	var err error
	if filters != nil && len(filters) > 0 {
		filteringJSON, err = json.Marshal(filters)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal filters")
		}
	}

	u, err := url.Parse(apiSecurityResultPath + "/" + scanID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse URL")
	}
	q := u.Query()
	if filters != nil && len(filters) > 0 {
		q.Set("filtering", string(filteringJSON))
	}
	q.Set("per_page", "100")
	if pageStr, ok := queryParam["page"]; ok && pageStr != "" {
		q.Set("page", pageStr)
	} else {
		q.Set("page", "1")
	}
	u.RawQuery = q.Encode()
	path := u.String()

	resp, err := SendHTTPRequest(http.MethodGet, path, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var page APISecPaginatedResult
		err = decoder.Decode(&page)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &page, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
