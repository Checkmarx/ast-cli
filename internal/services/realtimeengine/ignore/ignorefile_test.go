package ignore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ossEntry struct {
	PackageManager string `json:"PackageManager"`
	PackageName    string `json:"PackageName"`
	PackageVersion string `json:"PackageVersion"`
}

func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	list, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestLoad_EmptyFile_ReturnsEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "empty.json")
	require.NoError(t, os.WriteFile(p, []byte("   \n"), 0o600))
	list, err := Load(p)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestAppend_DeDupes(t *testing.T) {
	e := ossEntry{"npm", "lodash", "4.17.20"}

	list, added, err := Append(nil, e)
	require.NoError(t, err)
	assert.True(t, added)
	assert.Len(t, list, 1)

	list, added, err = Append(list, e)
	require.NoError(t, err)
	assert.False(t, added, "identical entry must not be added twice")
	assert.Len(t, list, 1)

	list, added, err = Append(list, ossEntry{"npm", "lodash", "4.17.21"})
	require.NoError(t, err)
	assert.True(t, added)
	assert.Len(t, list, 2)
}

func TestAppend_DeDupe_IgnoresKeyOrder(t *testing.T) {
	existing := json.RawMessage(`{"PackageVersion":"4.17.20","PackageName":"lodash","PackageManager":"npm"}`)
	list, added, err := Append([]json.RawMessage{existing}, ossEntry{"npm", "lodash", "4.17.20"})
	require.NoError(t, err)
	assert.False(t, added, "key order must not affect de-dupe")
	assert.Len(t, list, 1)
}

func TestRemove(t *testing.T) {
	e := ossEntry{"npm", "lodash", "4.17.20"}
	list, _, _ := Append(nil, e)

	list, removed, err := Remove(list, e)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Empty(t, list)

	_, removed, err = Remove(list, e)
	require.NoError(t, err)
	assert.False(t, removed, "removing a missing entry is a no-op")
}

func TestSaveLoad_RoundTrip_CreatesParentDir(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".checkmarx", ".checkmarxIgnoredTempList.json")
	list, _, _ := Append(nil, ossEntry{"npm", "lodash", "4.17.20"})
	require.NoError(t, Save(p, list))

	loaded, err := Load(p)
	require.NoError(t, err)
	require.Len(t, loaded, 1)

	var got ossEntry
	require.NoError(t, json.Unmarshal(loaded[0], &got))
	assert.Equal(t, "lodash", got.PackageName)
}

func TestDefaultPath(t *testing.T) {
	assert.Equal(t, filepath.Join(".checkmarx", "checkmarxIgnoredTempList.json"), DefaultPath())
}

func TestPathFor_AnchorsAtWorkDir(t *testing.T) {
	workDir := filepath.Join("some", "workspace")
	assert.Equal(t,
		filepath.Join(workDir, ".checkmarx", "checkmarxIgnoredTempList.json"),
		PathFor(workDir),
	)
}

func TestPathFor_EmptyWorkDirFallsBackToDefault(t *testing.T) {
	assert.Equal(t, DefaultPath(), PathFor(""))
}
