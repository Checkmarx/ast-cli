package wrappers

import rm "github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"

var StatResolutions = map[string]StatResolution{
	"moment": StatResolutionMoment,
	"minute": StatResolutionMinute,
	"hour":   StatResolutionHour,
	"day":    StatResolutionDay,
	"week":   StatResolutionWeek,
}

type StatResolution string

const (
	StatResolutionMoment = "moment"
	StatResolutionMinute = "minute"
	StatResolutionHour   = "hour"
	StatResolutionDay    = "day"
	StatResolutionWeek   = "week"
)

type SastRmWrapper interface {
	GetScans() ([]*rm.Scan, error)
	GetEngines() ([]*rm.Engine, error)
	GetStats(resolution StatResolution) ([]*rm.Metric, error)
}
