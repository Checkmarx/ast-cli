package util

import (
	"strings"
)

const (
	// KicsEngineEmbedded runs KICS in-process from the query assets compiled into the CLI.
	// It is the default because it needs no container runtime and no download.
	KicsEngineEmbedded = "embedded"
	// KicsEngineDocker names the Docker container engine.
	KicsEngineDocker = "docker"
)

// IsEmbeddedKicsEngine reports whether engine names the in-process KICS engine.
func IsEmbeddedKicsEngine(engine string) bool {
	return strings.EqualFold(strings.TrimSpace(engine), KicsEngineEmbedded)
}

// ResolveKicsEngine picks the KICS backend. An explicit engine name always wins, so callers
// that ask for docker or podman keep the container behaviour. With nothing specified the
// in-process engine is used, which is why an IaC scan no longer requires a container runtime.
func ResolveKicsEngine(engineFlag string) string {
	if engine := strings.TrimSpace(engineFlag); engine != "" && !IsEmbeddedKicsEngine(engine) {
		return engine
	}
	return KicsEngineEmbedded
}
