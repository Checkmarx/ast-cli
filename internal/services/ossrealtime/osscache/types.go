package osscache

import "time"

type PackageEntry struct {
	PackageManager string `json:"packageManager"`
	PackageName    string `json:"packageName"`
	PackageVersion string `json:"packageVersion"`
	Status         string `json:"status"`
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
