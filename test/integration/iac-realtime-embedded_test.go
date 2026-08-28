//go:build integration

package integration

import (
	"testing"

	guardrailskics "github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/kics"
	"github.com/checkmarx/ast-cli/internal/commands/util"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/stretchr/testify/assert"
)

// parityFixtures are the same inputs the container-only suite scans, so a parity pass means
// the two engines agree on findings the baseline suite already exercises.
var parityFixtures = []string{
	"data/positive1.tf",
	"data/positive/Dockerfile",
}

func runIacRealtime(t *testing.T, fixture, engine string) ([]iacrealtime.IacRealtimeResult, error) {
	t.Helper()
	err, out := executeCommand(t, "scan", "iac-realtime",
		flag(commonParams.SourcesFlag), fixture,
		flag(commonParams.EngineFlag), engine,
	)
	if err != nil {
		return nil, err
	}

	var results []iacrealtime.IacRealtimeResult
	assert.Nil(t, safeJSONUnmarshal(out.Bytes(), &results), "failed to unmarshal %s IAC results", engine)
	return results, nil
}

// TestIacRealtimeScan_EmbeddedEngine_NeedsNoContainerRuntime is the headline guarantee of the
// embedded backend: an IaC scan completes with no container engine present. It deliberately
// does not skip when Docker is missing - that is precisely the case it exists to cover.
func TestIacRealtimeScan_EmbeddedEngine_NeedsNoContainerRuntime(t *testing.T) {
	configuration.LoadConfiguration()

	for _, fixture := range parityFixtures {
		t.Run(fixture, func(t *testing.T) {
			_, err := runIacRealtime(t, fixture, util.KicsEngineEmbedded)
			assert.Nil(t, err, "embedded engine scan should succeed without any container runtime")
		})
	}
}

// TestIacRealtimeScan_EmbeddedEngine_IsDefault checks that omitting --engine selects the
// embedded backend, so plugins get the container-free path without changing how they invoke
// the CLI.
func TestIacRealtimeScan_EmbeddedEngine_IsDefault(t *testing.T) {
	configuration.LoadConfiguration()

	err, out := executeCommand(t, "scan", "iac-realtime",
		flag(commonParams.SourcesFlag), parityFixtures[0],
	)
	assert.Nil(t, err, "iac-realtime should succeed with no --engine flag")
	assert.NotNil(t, out, "iac-realtime should produce output with no --engine flag")
}

// TestIacRealtimeScan_EmbeddedParity_WithDocker is the regression gate for KICS upgrades.
// Both engines must report exactly the same findings, compared by Title and SimilarityID, so
// swapping the backend cannot change finding identity and therefore cannot break ignore-file
// matching in the IDE plugins.
func TestIacRealtimeScan_EmbeddedParity_WithDocker(t *testing.T) {
	configuration.LoadConfiguration()

	for _, fixture := range parityFixtures {
		t.Run(fixture, func(t *testing.T) {
			dockerResults, dockerErr := runIacRealtime(t, fixture, util.KicsEngineDocker)
			if dockerErr != nil {
				t.Skipf("container engine unavailable, cannot compare against it: %v", dockerErr)
			}

			embeddedResults, embeddedErr := runIacRealtime(t, fixture, util.KicsEngineEmbedded)
			assert.Nil(t, embeddedErr, "embedded engine scan should succeed")

			// NewFindings reports entries keyed by Title+SimilarityID present in the second
			// set but not the first; both directions empty means the engines agree exactly.
			assert.Empty(t, guardrailskics.NewFindings(dockerResults, embeddedResults),
				"findings reported only by the embedded engine for %s", fixture)
			assert.Empty(t, guardrailskics.NewFindings(embeddedResults, dockerResults),
				"findings reported only by the docker engine for %s", fixture)
		})
	}
}
