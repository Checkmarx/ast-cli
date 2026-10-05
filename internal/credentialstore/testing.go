package credentialstore

import "sync"

// Test-only seams for the default-resolver singleton. They live in the
// production package (not _test.go) because harnesses in other packages —
// commands, util, agenthooks/mcp, wrappers — must inject fakes, and Go cannot
// export symbols from test files to external test packages.

// ResetForTest discards the default resolver singleton; tests only.
func ResetForTest() {
	resolverOnce = sync.Once{}
	defaultResolver = nil
}

// SetDefaultResolverForTest installs r as the default resolver; tests only.
// The no-op Once marks initialization as done so a later Default() call cannot
// overwrite the injected resolver.
func SetDefaultResolverForTest(r *Resolver) {
	resolverOnce.Do(func() {})
	defaultResolver = r
}
