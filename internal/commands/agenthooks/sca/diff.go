package sca

import (
	"os"
	"path/filepath"

	"github.com/Checkmarx/manifest-parser/pkg/parser"
)

// AddedPackages returns the set of packages present in after but not in
// before, keyed by name+version. A version bump on an existing package is
// reported as added (its new version is new).
//
// manifestPath is the real path of the edited file — used only for its
// basename, so manifest-parser's factory picks the parser that matches the
// file being diffed (IsManifest already confirmed it recognises the name).
// Both before and after are parsed via that same parser. An empty/missing
// before parses as zero packages (so every after-package is "added").
func AddedPackages(manifestPath string, before, after []byte) ([]Package, error) {
	beforePkgs, err := parseManifestBytes(manifestPath, before)
	if err != nil {
		return nil, err
	}
	afterPkgs, err := parseManifestBytes(manifestPath, after)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(beforePkgs))
	for _, p := range beforePkgs {
		seen[pkgKey(p)] = struct{}{}
	}
	var added []Package
	for _, p := range afterPkgs {
		if _, ok := seen[pkgKey(p)]; ok {
			continue
		}
		added = append(added, p)
	}
	return added, nil
}

func parseManifestBytes(manifestPath string, content []byte) ([]Package, error) {
	if len(content) == 0 {
		return nil, nil
	}
	dir, err := os.MkdirTemp("", "sca-diff-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, filepath.Base(manifestPath))
	if writeErr := os.WriteFile(path, content, 0600); writeErr != nil {
		return nil, writeErr
	}

	p := parser.ParsersFactory(path)
	if p == nil {
		return nil, nil
	}
	rawPkgs, err := p.Parse(path)
	if err != nil {
		return nil, err
	}

	out := make([]Package, 0, len(rawPkgs))
	for _, rp := range rawPkgs {
		out = append(out, Package{Name: rp.PackageName, Version: rp.Version})
	}
	return out, nil
}

func pkgKey(p Package) string {
	return p.Name + "\x00" + p.Version
}
