// Package assetspec defines which KICS asset files the CLI embeds.
//
// It is a leaf package on purpose. The generator that builds the embedded archive and the
// test that guards the archive against drift both need this rule, and neither can take it
// from the kicsengine package itself: kicsengine embeds the archive, so it does not compile
// until the archive exists, which is exactly what the generator is there to produce.
package assetspec

import "path"

// Directory names under the KICS module's assets/ tree.
const (
	Root          = "assets"
	QueriesDir    = "queries"
	LibrariesDir  = "libraries"
	TransitionDir = "similarityID_transition"
)

// runtimeQueryFiles are the only files under assets/queries that the engine loads at scan
// time. The rest of that tree is per-query test fixtures, which would add roughly 45 MB to
// the CLI for nothing.
var runtimeQueryFiles = map[string]bool{
	"query.rego":    true,
	"metadata.json": true,
}

// Trees are the asset directories that get embedded, relative to the KICS module root.
func Trees() []string {
	return []string{
		path.Join(Root, QueriesDir),
		path.Join(Root, LibrariesDir),
		path.Join(Root, TransitionDir),
	}
}

// Include reports whether a file inside one of Trees() belongs in the embedded archive.
// tree is the entry from Trees() the file was found under; name is its base name.
func Include(tree, name string) bool {
	if tree == path.Join(Root, QueriesDir) {
		return runtimeQueryFiles[name]
	}
	return true
}
