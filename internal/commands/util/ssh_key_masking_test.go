//go:build !integration

package util

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

// Request bodies are dumped JSON-encoded, so a multi-line SSH key appears with
// escaped newlines and must still be masked in debug logs.
func TestSSHKeyIsMaskedInRequestDump(t *testing.T) {
	const sshKey = "-----BEGIN OPENSSH PRIVATE KEY-----\r\n" +
		"b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW\r\n" +
		"QyNTUxOQAAACDqaGl0aGVyZWZha2VrZXlmb3J0ZXN0aW5nb25seQAAAJgAAAAAAAAAAAAA\r\n" +
		"-----END OPENSSH PRIVATE KEY-----\n"
	viper.Set(params.SSHValue, sshKey)
	viper.Set(params.DebugFlag, true)
	defer viper.Reset()

	body, err := json.Marshal(map[string]string{"key": "scan.handler.git.sshKey", "value": sshKey})
	assert.NilError(t, err)
	req, err := http.NewRequest(http.MethodPatch, "http://localhost/api/configuration/project", bytes.NewReader(body))
	assert.NilError(t, err)

	var buf strings.Builder
	original := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(original)

	logger.PrintRequest(req)

	out := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(sshKey), "\n") {
		line = strings.TrimSpace(line)
		assert.Assert(t, !strings.Contains(out, line), "ssh key line leaked to logs: %s", out)
	}
	assert.Assert(t, strings.Contains(out, `"value":"***"`), "expected masked ssh key in log output: %s", out)
}
