package osscache

import (
	"time"
)

type PackageEntry struct {
	PackageManager  string          `json:"packageManager"`
	PackageName     string          `json:"packageName"`
	PackageVersion  string          `json:"packageVersion"`
	Status          string          `json:"status"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

type Vulnerability struct {
	CVE         string `json:"CVE"`
	Description string `json:"Description"`
	Severity    string `json:"Severity"`
}

type Cache struct {
	TTL      time.Time      `json:"ttl"`
	Packages []PackageEntry `json:"packages"`
}

func (c *Cache) GetTTL() time.Time {
	return c.TTL
}

func (c *Cache) SetTTL(t time.Time) {
	c.TTL = t
}
