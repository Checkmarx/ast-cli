package kicsengine

import (
	"path/filepath"
	"strings"
)

// Platform names are KICS's internal identifiers, the ones CheckType compares against in
// pkg/engine/source/filesystem.go. They double as the directory names under assets/queries,
// which is what lets queryDirs point the query source at a subtree.
const (
	platformTerraform  = "terraform"
	platformDockerfile = "dockerfile"
	platformAnsible    = "ansible"
	platformGRPC       = "grpc"
	platformBuildah    = "buildah"

	commonQueriesDir = "common"
)

// singlePlatformExtensions maps a file extension to the one platform that can match it.
//
// Every entry was checked against the parser registrations in KICS pkg/parser: each of these
// extensions is claimed by exactly one parser, and that parser reports exactly one platform.
// Extensions claimed by more than one parser are deliberately absent - see PlatformsForFile.
var singlePlatformExtensions = map[string]string{
	".tf":     platformTerraform,
	".tfvars": platformTerraform,

	".dockerfile": platformDockerfile,
	".ubi8":       platformDockerfile,
	".debian":     platformDockerfile,

	".proto": platformGRPC,
	".sh":    platformBuildah,

	".ini":  platformAnsible,
	".cfg":  platformAnsible,
	".conf": platformAnsible,
}

// PlatformsForFile returns the KICS platforms worth loading for path, or nil to load every
// platform.
//
// Narrowing matters because KICS compiles each query's Rego at inspection time rather than
// once up front, so scan cost is close to linear in the number of queries loaded: the full
// set is ~1810 queries, Terraform alone is ~762, Dockerfile ~48.
//
// It only narrows on extensions that map to exactly one platform. Ambiguous ones deliberately
// return nil: a .yaml file could be Kubernetes, Ansible, CloudFormation, OpenAPI, Crossplane,
// Knative, Pulumi, ServerlessFW or DockerCompose, and guessing wrong would mean loading the
// wrong parser and silently reporting no findings. Loading everything is slower but correct,
// which is the right default for a security scanner.
//
// .bicep is also excluded even though it has a single parser: its queries live under the
// azureResourceManager directory while the parser reports the "bicep" platform, so narrowing
// it would find no queries at all.
//
// Queries on the "common" platform are always loaded by KICS regardless of this filter, so
// narrowing never drops shared logic.
func PlatformsForFile(path string) []string {
	name := filepath.Base(path)
	if platform, ok := singlePlatformExtensions[strings.ToLower(filepath.Ext(name))]; ok {
		return []string{platform}
	}

	// Dockerfile, dockerfile, Dockerfile.prod - all recognised by KICS's Dockerfile parser.
	if lower := strings.ToLower(name); lower == platformDockerfile || strings.HasPrefix(lower, "dockerfile.") {
		return []string{platformDockerfile}
	}

	return nil
}

// queryDirs returns the directories the query source should read, given the narrowed platform
// set. Platform names double as directory names under assets/queries.
//
// This is what turns narrowing into an I/O saving rather than only a CPU one: KICS's
// FilesystemSource walks every directory it is given and reads query.rego plus metadata.json
// from all ~1810 of them before discarding the wrong platforms, so pointing it at a subtree
// avoids reading roughly 12 MB for a scan that needs a fraction of it.
//
// The common directory is always included; it holds no query.rego today, but including it
// keeps the behaviour correct if that ever changes.
func queryDirs(queriesRoot string, platforms []string) []string {
	if len(platforms) == 0 {
		return []string{queriesRoot}
	}

	dirs := make([]string, 0, len(platforms)+1)
	for _, platform := range platforms {
		dirs = append(dirs, filepath.Join(queriesRoot, platform))
	}
	return append(dirs, filepath.Join(queriesRoot, commonQueriesDir))
}
