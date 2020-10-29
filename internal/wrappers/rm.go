package wrappers

import rm "github.com/checkmarxDev/sast-rm/pkg/api/rest"

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
	GetPools() ([]*rm.Pool, error)
	AddPool(description string) (*rm.Pool, error)
	DeletePool(id string) error
	GetPoolEngines(id string) ([]string, error)
	GetPoolProjects(id string) ([]string, error)
	GetPoolEngineTags(id string) ([]Tag, error)
	GetPoolProjectTags(id string) ([]Tag, error)
	SetPoolEngines(id string, value []string) error
	SetPoolProjects(id string, value []string) error
	SetPoolEngineTags(id string, tags []Tag) error
	SetPoolProjectTags(id string, tags []Tag) error
	GetStats(resolution StatResolution) ([]*rm.Metric, error)
}

type Tag struct {
	Key, Value string
}
