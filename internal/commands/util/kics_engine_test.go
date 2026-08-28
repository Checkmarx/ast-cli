//go:build !integration

package util

import (
	"testing"

	"gotest.tools/assert"
)

func TestResolveKicsEngine(t *testing.T) {
	tests := []struct {
		name string
		flag string
		want string
	}{
		{name: "empty defaults to the embedded engine", flag: "", want: KicsEngineEmbedded},
		{name: "embedded flag is case insensitive", flag: "EMBEDDED", want: KicsEngineEmbedded},
		{name: "embedded flag is trimmed", flag: "  embedded  ", want: KicsEngineEmbedded},
		{name: "docker flag passes through", flag: "docker", want: KicsEngineDocker},
		{name: "container name passes through", flag: "podman", want: "podman"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ResolveKicsEngine(tt.flag), tt.want)
		})
	}
}

func TestIsEmbeddedKicsEngine(t *testing.T) {
	for _, engine := range []string{"embedded", "EMBEDDED", " Embedded "} {
		assert.Assert(t, IsEmbeddedKicsEngine(engine), "expected %q to name the embedded engine", engine)
	}
	for _, engine := range []string{"", "docker", "podman"} {
		assert.Assert(t, !IsEmbeddedKicsEngine(engine), "expected %q not to name the embedded engine", engine)
	}
}
