package kicsengine

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/kicsengine/assetspec"
	"gotest.tools/assert"
)

// TestEmbeddedAssetsMatchPinnedModule is the drift guard.
//
// The embedded archive is generated from whichever KICS version go.mod pins. If someone bumps
// that version without re-running "go generate ./internal/services/kicsengine/...", the CLI
// would link new KICS code against a stale query set - findings and SimilarityIDs would move
// silently, with nothing failing. This walks the pinned module with the same assetspec rule
// the generator uses and compares it against what was actually extracted.
func TestEmbeddedAssetsMatchPinnedModule(t *testing.T) {
	moduleDir := pinnedModuleDir(t)

	root, err := AssetsRoot()
	assert.NilError(t, err)

	embedded := extractedFiles(t, root)
	expected := specifiedFiles(t, moduleDir)

	assert.Equal(t, len(embedded), len(expected),
		"embedded asset count differs from the pinned KICS module; run go generate ./internal/services/kicsengine/...")

	for i := range expected {
		assert.Equal(t, embedded[i], expected[i],
			"embedded assets diverge from the pinned KICS module; run go generate ./internal/services/kicsengine/...")
	}
}

// TestSimilarityTransitionAssetsPresent pins the behaviour that actually breaks users when it
// regresses: without these files KICS silently stops applying similarity-ID transitions, which
// changes finding identity and therefore breaks ignore-file matching.
func TestSimilarityTransitionAssetsPresent(t *testing.T) {
	root, err := AssetsRoot()
	assert.NilError(t, err)

	entries, err := os.ReadDir(filepath.Join(root, assetsDirName, transitionDirName))
	assert.NilError(t, err)
	assert.Assert(t, len(entries) > 0, "similarity-ID transition assets must be embedded")
}

func pinnedModuleDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", kicsModulePath).Output()
	if err != nil {
		t.Skipf("pinned KICS module not available in the module cache: %v", err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		t.Skip("pinned KICS module has no local directory")
	}
	return dir
}

// extractedFiles lists everything under the extracted assets directory. The completion marker
// sits beside that directory rather than inside it, so it is naturally excluded.
func extractedFiles(t *testing.T, installRoot string) []string {
	t.Helper()
	return walkRelative(t, filepath.Join(installRoot, assetsDirName), nil)
}

// specifiedFiles lists what assetspec says the archive should contain, walking only the trees
// it names - the KICS module's assets directory also holds cwe_csv and template, which are not
// embedded.
func specifiedFiles(t *testing.T, moduleDir string) []string {
	t.Helper()

	var out []string
	for _, tree := range assetspec.Trees() {
		keep := func(name string) bool { return assetspec.Include(tree, name) }
		for _, rel := range walkRelative(t, filepath.Join(moduleDir, filepath.FromSlash(tree)), keep) {
			// Re-root onto the assets directory so both sides use the same prefix.
			out = append(out, strings.TrimPrefix(tree, assetspec.Root+"/")+"/"+rel)
		}
	}
	sort.Strings(out)
	return out
}

// walkRelative returns slash-separated paths under root, sorted, optionally filtered by base name.
func walkRelative(t *testing.T, root string, keep func(name string) bool) []string {
	t.Helper()

	var out []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if keep != nil && !keep(info.Name()) {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	assert.NilError(t, err)
	sort.Strings(out)
	return out
}
