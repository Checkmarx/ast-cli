package wrappers

import rm "github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"

type StatMetric string

const (
	StatMetricScanQueued    = "scan-queued"
	StatMetricScanOrphan    = "scan-orphan"
	StatMetricEngineWaiting = "engine-waiting"
	StatMetricRunning       = "running"
)

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
