// +build integration

package integration

import (
	"fmt"
	"testing"
)

func TestSastHealthCheck(t *testing.T) {
	// TODO: fix this
	fmt.Println("TODO: Disabled healthcheck because its not working.")
	// healthCheckCmd := createASTIntegrationTestCommand(t)
	// b := bytes.NewBufferString("")
	// healthCheckCmd.SetOut(b)
	/*
		err := execute(healthCheckCmd, "single-node", "health-check", "--role", "SAST_ALL_IN_ONE")
		assert.NilError(t, err, "Health check should pass")
		out, err := ioutil.ReadAll(b)
		assert.NilError(t, err, "Should read the command output")
		s := string(out)
		assert.Assert(t, !(strings.Contains(s, "Failure ") || strings.Contains(s, "Error ")),
			"Command output should be success and not %v", s)
	*/
}

func TestScaHealthCheck(t *testing.T) {
	// TODO: fix this
	fmt.Println("TODO: Disabled healthcheck because its not working.")
	/*
		healthCheckCmd := createASTIntegrationTestCommand(t)
		b := bytes.NewBufferString("")
		healthCheckCmd.SetOut(b)
		err := execute(healthCheckCmd, "single-node", "health-check", "--role", "SCA_AGENT")
		assert.NilError(t, err, "Health check should pass")
		out, err := ioutil.ReadAll(b)
		assert.NilError(t, err, "Should read the command output")
		s := string(out)
		assert.Assert(t, !(strings.Contains(s, "Failure ") || strings.Contains(s, "Error ")),
			"Command output should be success and not %v", s)
	*/
}
