// +build integration

package integration

import (
	"fmt"
	"github.com/checkmarxDev/ast-cli/internal/params"
	"gotest.tools/assert"
	"io/ioutil"
	"strings"
	"testing"
)

var roles = []string{
	params.ScaAgent,
	params.SastEngine,
	params.SastALlInOne,
}

func TestHealthCheck(t *testing.T) {
	for _, role := range roles {
		fmt.Printf("Health check test for role %s\n", role)
		executeHealthCheckTest(t, role)
	}
}

func executeHealthCheckTest(t *testing.T, role string) {
	healthCheckCommand, outBuffer := createRedirectedTestCommand(t)

	err := execute(healthCheckCommand, "utils", "health-check", "--role", role)
	assert.NilError(t, err, "Health check for role %s should pass", role)

	out, err := ioutil.ReadAll(outBuffer)
	assert.NilError(t, err, "Should read the command output for role %s")

	outputStr := strings.ToLower(string(out))

	assert.Assert(t, !strings.Contains(outputStr, "error"),
		"Command output for role %s should not contain error", role)
	assert.Assert(t, !strings.Contains(outputStr, "fail"),
		"Command output for role %s should not contain fail", role)
}
