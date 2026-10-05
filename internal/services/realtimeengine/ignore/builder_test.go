package ignore

import (
	"encoding/json"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func entryMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func TestBuildEntries_OSS_DropsExtraScanFields(t *testing.T) {
	finding := `{"PackageManager":"npm","PackageName":"lodash","PackageVersion":"4.17.20",
		"FilePath":"package.json","Vulnerabilities":[{"CVE":"CVE-2021-23337","Severity":"High"}]}`
	entries, err := BuildEntries("oss", []byte(finding))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	m := entryMap(t, entries[0])
	assert.Equal(t, "npm", m["PackageManager"])
	assert.Equal(t, "lodash", m["PackageName"])
	assert.Equal(t, "4.17.20", m["PackageVersion"])
	_, hasVulns := m["Vulnerabilities"]
	assert.False(t, hasVulns, "only key fields should be persisted")
}

func TestBuildEntries_SCAAlias(t *testing.T) {
	entries, err := BuildEntries("sca", []byte(`{"PackageManager":"npm","PackageName":"lodash","PackageVersion":"4.17.20"}`))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestBuildEntries_Secrets(t *testing.T) {
	entries, err := BuildEntries("secrets", []byte(`{"Title":"github-pat","SecretValue":"ghp_x","FilePath":"a.js","Severity":"High"}`))
	require.NoError(t, err)
	m := entryMap(t, entries[0])
	assert.Equal(t, "github-pat", m["Title"])
	assert.Equal(t, "ghp_x", m["SecretValue"])
}

func TestBuildEntries_Containers(t *testing.T) {
	entries, err := BuildEntries("containers", []byte(`{"ImageName":"ubuntu","ImageTag":"14.04","Status":"OK"}`))
	require.NoError(t, err)
	m := entryMap(t, entries[0])
	assert.Equal(t, "ubuntu", m["ImageName"])
	assert.Equal(t, "14.04", m["ImageTag"])
}

func TestBuildEntries_IaC(t *testing.T) {
	entries, err := BuildEntries("iac", []byte(`{"Title":"Missing User Instruction","SimilarityID":"abc123","Severity":"High"}`))
	require.NoError(t, err)
	m := entryMap(t, entries[0])
	assert.Equal(t, "Missing User Instruction", m["Title"])
	assert.Equal(t, "abc123", m["SimilarityID"])
}

// The critical ASCA case: scan output is snake_case (file_name/line/rule_id), the ignore entry is
// PascalCase (FileName/Line/RuleID). A naive direct unmarshal would silently lose FileName/RuleID.
func TestBuildEntries_ASCA_MapsSnakeCaseScanOutput(t *testing.T) {
	finding := `{"rule_id":5004,"rule_name":"Insecure Logging","file_name":"server.py","line":77,
		"problematicLine":"log(pw)","severity":"High"}`
	entries, err := BuildEntries("asca", []byte(finding))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	ig, ok := entries[0].(grpcs.AscaIgnoreFinding)
	require.True(t, ok)
	assert.Equal(t, "server.py", ig.FileName)
	assert.Equal(t, uint32(77), ig.Line)
	assert.Equal(t, uint32(5004), ig.RuleID)

	m := entryMap(t, entries[0])
	assert.Equal(t, "server.py", m["FileName"])
	assert.EqualValues(t, 5004, m["RuleID"])
}

func TestBuildEntries_ASCA_AcceptsIgnoreEntryShape(t *testing.T) {
	entries, err := BuildEntries("asca", []byte(`{"FileName":"server.py","Line":77,"RuleID":5004}`))
	require.NoError(t, err)
	ig := entries[0].(grpcs.AscaIgnoreFinding)
	assert.Equal(t, "server.py", ig.FileName)
	assert.Equal(t, uint32(5004), ig.RuleID)
}

func TestBuildEntries_FullPayloadWrappers(t *testing.T) {
	ossEntries, err := BuildEntries("oss", []byte(`{"Packages":[
		{"PackageManager":"npm","PackageName":"a","PackageVersion":"1"},
		{"PackageManager":"npm","PackageName":"b","PackageVersion":"2"}]}`))
	require.NoError(t, err)
	assert.Len(t, ossEntries, 2)

	ascaEntries, err := BuildEntries("asca", []byte(`{"scan_details":[{"file_name":"x.py","line":1,"rule_id":10}]}`))
	require.NoError(t, err)
	assert.Len(t, ascaEntries, 1)

	secretEntries, err := BuildEntries("secrets", []byte(`[{"Title":"t","SecretValue":"s"}]`))
	require.NoError(t, err)
	assert.Len(t, secretEntries, 1)
}

func TestBuildEntries_MissingRequiredFields_Errors(t *testing.T) {
	_, err := BuildEntries("oss", []byte(`{"PackageManager":"npm","PackageName":"lodash"}`))
	require.Error(t, err)

	_, err = BuildEntries("asca", []byte(`{"file_name":"server.py","line":77}`)) // no rule_id
	require.Error(t, err)
}

func TestBuildEntries_BadJSON_Errors(t *testing.T) {
	_, err := BuildEntries("oss", []byte(`not json`))
	require.Error(t, err)
}

func TestIsValidScanType(t *testing.T) {
	for _, s := range []string{"oss", "sca", "secrets", "containers", "iac", "asca", "OSS", "ASCA", " sca "} {
		assert.Truef(t, IsValidScanType(s), "expected %q valid", s)
	}
	for _, s := range []string{"", "foo", "sast", "kics"} {
		assert.Falsef(t, IsValidScanType(s), "expected %q invalid", s)
	}
}
