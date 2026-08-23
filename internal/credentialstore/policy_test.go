package credentialstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePolicy(t *testing.T) {
	cases := []struct {
		raw     string
		want    Policy
		wantErr bool
	}{
		{raw: "", want: PolicyAuto},
		{raw: "auto", want: PolicyAuto},
		{raw: "required", want: PolicyRequired},
		{raw: "disabled", want: PolicyDisabled},
		{raw: "AUTO", wantErr: true},
		{raw: "bogus", wantErr: true},
	}
	for _, tc := range cases {
		got, err := ParsePolicy(tc.raw)
		if tc.wantErr {
			assert.Error(t, err, tc.raw)
			continue
		}
		assert.NoError(t, err, tc.raw)
		assert.Equal(t, tc.want, got, tc.raw)
	}
}

func TestParsePolicyInvalidListsValidValues(t *testing.T) {
	_, err := ParsePolicy("bogus")
	assert.ErrorContains(t, err, policyValueAuto)
	assert.ErrorContains(t, err, policyValueRequired)
	assert.ErrorContains(t, err, policyValueDisabled)
}

func TestPolicyFromEnv(t *testing.T) {
	t.Setenv(KeyringModeEnvVar, "required")
	assert.Equal(t, PolicyRequired, PolicyFromEnv())

	t.Setenv(KeyringModeEnvVar, "disabled")
	assert.Equal(t, PolicyDisabled, PolicyFromEnv())
}

func TestPolicyFromEnvUnsetDefaultsToAuto(t *testing.T) {
	t.Setenv(KeyringModeEnvVar, "")
	assert.Equal(t, PolicyAuto, PolicyFromEnv())
}

func TestPolicyFromEnvInvalidFallsBackToAuto(t *testing.T) {
	t.Setenv(KeyringModeEnvVar, "sometimes")
	assert.Equal(t, PolicyAuto, PolicyFromEnv())
}
