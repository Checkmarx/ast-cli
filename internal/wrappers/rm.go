package wrappers

import rm "github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"

type StatMetric string

const (
	StatMetricScanPending   = "scan-pending"
	StatMetricScanRunning   = "scan-running"
	StatMetricScanOrphan    = "scan-orphan"
	StatMetricScanTotal     = "scan-total"
	StatMetricEngineWaiting = "engine-waiting"
	StatMetricEngineRunning = "engine-running"
	StatMetricEngineTotal   = "engine-total"
)

var StatMetrics = map[string]StatMetric{
	"scan-pending":   StatMetricScanPending,
	"scan-running":   StatMetricScanRunning,
	"scan-orphan":    StatMetricScanOrphan,
	"scan-total":     StatMetricScanTotal,
	"engine-waiting": StatMetricEngineWaiting,
	"engine-running": StatMetricEngineRunning,
	"engine-total":   StatMetricEngineTotal,
}

var StatResolutions = map[string]StatResolution{
	"hour": StatResolutionHour,
	"day":  StatResolutionDay,
	"week": StatResolutionWeek,
}

type StatResolution string

const (
	StatResolutionHour = "hour"
	StatResolutionDay  = "day"
	StatResolutionWeek = "week"
)

type SastRmWrapper interface {
	GetScans() ([]*rm.Scan, error)
	GetEngines() ([]*rm.Engine, error)
	GetStats(metric StatMetric, resolution StatResolution) ([]*rm.Counter, error)
}
