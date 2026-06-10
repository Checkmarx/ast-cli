package sca

import (
	"path/filepath"
	"strings"
)

// Manager identifies a package manager whose install commands the Bash hook
// recognises.
type Manager int

const (
	ManagerUnknown Manager = iota
	ManagerNpm
	ManagerPypi
	ManagerDotnet
	ManagerGo
	ManagerMaven
)

// Format returns the manifest format that pairs with this manager — used to
// synthesise a temp manifest for oss-realtime.
func (m Manager) Format() Format {
	switch m {
	case ManagerNpm:
		return FormatNpmPackageJson
	case ManagerPypi:
		return FormatPypiRequirements
	case ManagerDotnet:
		return FormatDotnetCsproj
	case ManagerGo:
		return FormatGoMod
	case ManagerMaven:
		return FormatMavenPom
	}
	return FormatUnknown
}

// Package is a parsed install target.
type Package struct {
	Name    string
	Version string // "" → unspecified (oss-realtime defaults to "latest")
}

// InstallRequest is one recognised install invocation extracted from a shell
// command. A compound command may produce multiple requests.
type InstallRequest struct {
	Manager     Manager
	Packages    []Package
	ManifestRef string // set for `pip install -r <file>` — Packages is empty
}

// ParseInstall returns every recognised install invocation in command, or
// nil if none. Compound commands (&&, ||, ;, |) are split before matching, so
// `cd /repo && npm install lodash` returns one npm request, and
// `npm install lodash && pip install requests` returns two.
func ParseInstall(command string) []InstallRequest {
	var requests []InstallRequest
	for _, segment := range splitTopLevel(command) {
		if req := parseSegment(segment); req != nil {
			requests = append(requests, *req)
		}
		// Also descend into $(...) and `...` subshells inside the segment.
		for _, sub := range extractSubshells(segment) {
			requests = append(requests, ParseInstall(sub)...)
		}
	}
	return requests
}

// splitTopLevel splits command on top-level shell operators (&&, ||, ;, |),
// honouring single/double quotes and paren/$()/backtick nesting so we don't
// split inside string literals or subshells.
func splitTopLevel(command string) []string {
	var (
		segments []string
		current  strings.Builder
		sq, dq   bool
		paren    int // ( and $( nesting depth
		bt       int // backtick nesting (0 or 1 — backticks don't nest in practice)
	)
	flush := func() {
		seg := strings.TrimSpace(current.String())
		if seg != "" {
			segments = append(segments, seg)
		}
		current.Reset()
	}
	for i := 0; i < len(command); i++ {
		c := command[i]
		switch {
		case sq:
			if c == '\'' {
				sq = false
			}
			current.WriteByte(c)
		case dq:
			if c == '"' {
				dq = false
			} else if c == '\\' && i+1 < len(command) {
				current.WriteByte(c)
				i++
				current.WriteByte(command[i])
				continue
			}
			current.WriteByte(c)
		case c == '\'':
			sq = true
			current.WriteByte(c)
		case c == '"':
			dq = true
			current.WriteByte(c)
		case c == '`':
			if bt == 0 {
				bt = 1
			} else {
				bt = 0
			}
			current.WriteByte(c)
		case bt > 0:
			current.WriteByte(c)
		case c == '$' && i+1 < len(command) && command[i+1] == '(':
			paren++
			current.WriteByte(c)
			i++
			current.WriteByte(command[i]) // the '('
		case c == '(':
			paren++
			current.WriteByte(c)
		case c == ')':
			if paren > 0 {
				paren--
			}
			current.WriteByte(c)
		case paren > 0:
			current.WriteByte(c)
		case c == '&' && i+1 < len(command) && command[i+1] == '&':
			flush()
			i++
		case c == '|' && i+1 < len(command) && command[i+1] == '|':
			flush()
			i++
		case c == ';':
			flush()
		case c == '|':
			flush()
		default:
			current.WriteByte(c)
		}
	}
	flush()
	return segments
}

// extractSubshells pulls out the bodies of $(...) and `...` subshells inside a
// segment so we can recursively parse them. Nested $() is respected.
func extractSubshells(segment string) []string {
	var bodies []string
	for i := 0; i < len(segment); i++ {
		c := segment[i]
		if c == '$' && i+1 < len(segment) && segment[i+1] == '(' {
			depth := 1
			j := i + 2
			for j < len(segment) && depth > 0 {
				switch segment[j] {
				case '(':
					depth++
				case ')':
					depth--
				}
				if depth == 0 {
					break
				}
				j++
			}
			if j > i+2 {
				bodies = append(bodies, segment[i+2:j])
			}
			i = j
		} else if c == '`' {
			j := i + 1
			for j < len(segment) && segment[j] != '`' {
				j++
			}
			if j > i+1 && j < len(segment) {
				bodies = append(bodies, segment[i+1:j])
			}
			i = j
		}
	}
	return bodies
}

// parseSegment recognises a single (non-compound) install command. Returns
// nil if the segment is not an install invocation.
func parseSegment(segment string) *InstallRequest {
	tokens := tokenize(segment)
	tokens = dropLeadingNoOps(tokens)
	if len(tokens) == 0 {
		return nil
	}

	mgr, rest := matchManager(tokens)
	if mgr == ManagerUnknown {
		return nil
	}

	switch mgr {
	case ManagerNpm:
		return parseNpmArgs(rest)
	case ManagerPypi:
		return parsePypiArgs(rest)
	case ManagerDotnet:
		return parseDotnetArgs(tokens, rest)
	case ManagerGo:
		return parseGoArgs(rest)
	case ManagerMaven:
		return parseMavenArgs(rest)
	}
	return nil
}

// tokenize splits a command into whitespace-separated tokens, preserving
// quoted strings, $(...) subshells, and `...` backticks as single tokens so
// downstream parsing isn't confused by internal whitespace inside shell
// expansions. Surrounding single/double quotes are stripped; subshell
// markers ($(, ), `) are preserved verbatim so callers can recognise them
// and skip those tokens.
func tokenize(segment string) []string {
	var (
		tokens   []string
		cur      strings.Builder
		sq, dq   bool
		paren    int
		inBT     bool
	)
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(segment); i++ {
		c := segment[i]
		switch {
		case sq:
			if c == '\'' {
				sq = false
			} else {
				cur.WriteByte(c)
			}
		case dq:
			if c == '"' {
				dq = false
			} else {
				cur.WriteByte(c)
			}
		case inBT:
			cur.WriteByte(c)
			if c == '`' {
				inBT = false
			}
		case c == '\'':
			sq = true
		case c == '"':
			dq = true
		case c == '`':
			cur.WriteByte(c)
			inBT = true
		case c == '$' && i+1 < len(segment) && segment[i+1] == '(':
			paren++
			cur.WriteByte(c)
			i++
			cur.WriteByte(segment[i]) // '('
		case paren > 0:
			cur.WriteByte(c)
			if c == ')' {
				paren--
			} else if c == '(' {
				paren++
			}
		case c == ' ' || c == '\t' || c == '\n':
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return tokens
}

// isShellExpansion reports whether tok is a $(...) or `...` expansion that
// should be skipped during package parsing (we can't statically know what
// the expansion evaluates to).
func isShellExpansion(tok string) bool {
	return strings.HasPrefix(tok, "$(") || strings.HasPrefix(tok, "`")
}

// dropLeadingNoOps strips command prefixes that don't change which install
// command runs: `sudo`, `time`, `nice`, env-style `FOO=bar`, and a leading
// `cd <dir>` chain (though `cd && ...` is split out at the operator level —
// this catches `(cd /foo; ...)`-style stripped patterns and standalone
// `bash -c "<cmd>"` already-unwrapped by the segment split).
func dropLeadingNoOps(tokens []string) []string {
	for len(tokens) > 0 {
		t := tokens[0]
		switch {
		case t == "sudo", t == "time", t == "nice":
			tokens = tokens[1:]
		case strings.Contains(t, "=") && !strings.HasPrefix(t, "-") && isEnvAssignment(t):
			tokens = tokens[1:]
		default:
			return tokens
		}
	}
	return tokens
}

func isEnvAssignment(t string) bool {
	idx := strings.Index(t, "=")
	if idx <= 0 {
		return false
	}
	for i := 0; i < idx; i++ {
		c := t[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// matchManager finds the install verb at the head of tokens (after no-op
// stripping). Returns (Manager, remaining tokens after the verb).
func matchManager(tokens []string) (Manager, []string) {
	if len(tokens) < 2 {
		return ManagerUnknown, nil
	}
	t0 := tokens[0]

	// Multi-token verbs first.
	if len(tokens) >= 3 && (t0 == "python" || t0 == "python3") && tokens[1] == "-m" && tokens[2] == "pip" {
		if len(tokens) >= 4 && tokens[3] == "install" {
			return ManagerPypi, tokens[4:]
		}
		return ManagerUnknown, nil
	}
	if t0 == "uv" && tokens[1] == "pip" {
		if len(tokens) >= 3 && tokens[2] == "install" {
			return ManagerPypi, tokens[3:]
		}
		return ManagerUnknown, nil
	}
	if t0 == "dotnet" && tokens[1] == "add" {
		if len(tokens) >= 3 && tokens[2] == "package" {
			return ManagerDotnet, tokens[3:]
		}
		return ManagerUnknown, nil
	}

	// Single-verb forms.
	verb := tokens[1]
	switch t0 {
	case "npm":
		if verb == "install" || verb == "i" || verb == "add" {
			return ManagerNpm, tokens[2:]
		}
	case "yarn":
		if verb == "add" {
			return ManagerNpm, tokens[2:]
		}
	case "pnpm":
		if verb == "add" || verb == "install" || verb == "i" {
			return ManagerNpm, tokens[2:]
		}
	case "pip", "pip3":
		if verb == "install" {
			return ManagerPypi, tokens[2:]
		}
	case "pipenv":
		if verb == "install" {
			return ManagerPypi, tokens[2:]
		}
	case "poetry":
		if verb == "add" {
			return ManagerPypi, tokens[2:]
		}
	case "uv":
		if verb == "add" {
			return ManagerPypi, tokens[2:]
		}
	case "nuget":
		if verb == "install" {
			return ManagerDotnet, tokens[2:]
		}
	case "go":
		if verb == "get" || verb == "install" {
			return ManagerGo, tokens[2:]
		}
	case "mvn":
		if verb == "dependency:get" {
			return ManagerMaven, tokens[2:]
		}
	}
	return ManagerUnknown, nil
}

// --- per-manager argument parsing ---

func parseNpmArgs(args []string) *InstallRequest {
	var pkgs []Package
	for _, a := range args {
		if skipArg(a) {
			continue
		}
		pkgs = append(pkgs, parseNpmSpec(a))
	}
	if len(pkgs) == 0 {
		return nil
	}
	return &InstallRequest{Manager: ManagerNpm, Packages: pkgs}
}

// parseNpmSpec handles bare names, name@version, scoped packages (@scope/pkg
// and @scope/pkg@version).
func parseNpmSpec(spec string) Package {
	if strings.HasPrefix(spec, "@") {
		rest := spec[1:]
		idx := strings.Index(rest, "@")
		if idx < 0 {
			return Package{Name: spec}
		}
		return Package{Name: "@" + rest[:idx], Version: normalizeSemver(rest[idx+1:])}
	}
	idx := strings.LastIndex(spec, "@")
	if idx <= 0 {
		return Package{Name: spec}
	}
	return Package{Name: spec[:idx], Version: normalizeSemver(spec[idx+1:])}
}

// normalizeSemver pads a bare numeric version with missing segments so the
// SCA scanner can look it up — e.g. "4.10" → "4.10.0", "4" → "4.0.0".
// Versions that contain non-numeric characters (ranges, pre-releases) are
// returned unchanged.
func normalizeSemver(v string) string {
	if v == "" {
		return v
	}
	parts := strings.Split(v, ".")
	if len(parts) >= 3 {
		return v
	}
	for _, p := range parts {
		for _, c := range p {
			if c < '0' || c > '9' {
				return v
			}
		}
	}
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	return strings.Join(parts, ".")
}

// parsePypiArgs handles bare names, name==ver, name>=ver style, and -r/-c
// requirement-file references.
func parsePypiArgs(args []string) *InstallRequest {
	var (
		pkgs []Package
		refs []string
		skip bool
	)
	for i, a := range args {
		if skip {
			skip = false
			continue
		}
		switch a {
		case "-r", "--requirement", "-c", "--constraint":
			if i+1 < len(args) {
				refs = append(refs, args[i+1])
				skip = true
			}
			continue
		case "-e", "--editable":
			// editable install of a local path — skip the next arg (the path).
			skip = true
			continue
		}
		if skipArg(a) {
			continue
		}
		pkgs = append(pkgs, parsePypiSpec(a))
	}
	// We collapse to a single request here. Multi-ref pip (-r a.txt -r b.txt)
	// only emits the first ref; multi-ref is rare and supporting it cleanly
	// would require parseSegment to return []*InstallRequest.
	if len(refs) > 0 {
		return &InstallRequest{Manager: ManagerPypi, ManifestRef: refs[0]}
	}
	if len(pkgs) > 0 {
		return &InstallRequest{Manager: ManagerPypi, Packages: pkgs}
	}
	return nil
}

func parsePypiSpec(spec string) Package {
	// Strip any comparator/version specifier; keep only exact-match versions.
	// requirements.txt convention: pkg==ver. Anything else → version unknown.
	for _, op := range []string{"==", ">=", "<=", "~=", "!=", ">", "<"} {
		if idx := strings.Index(spec, op); idx >= 0 {
			name := spec[:idx]
			if op == "==" {
				return Package{Name: name, Version: spec[idx+2:]}
			}
			return Package{Name: name}
		}
	}
	return Package{Name: spec}
}

// parseDotnetArgs supports both `dotnet add package <Name> [-v <ver>]` and
// `nuget install <Name> [-Version <ver>]`. tokens is the full token slice so
// we can decide which verb was used.
func parseDotnetArgs(tokens, args []string) *InstallRequest {
	isNuget := len(tokens) > 0 && tokens[0] == "nuget"

	var (
		name, version string
		skip          bool
	)
	for i, a := range args {
		if skip {
			skip = false
			continue
		}
		switch {
		case a == "-v" || a == "--version" || (isNuget && (a == "-Version" || a == "-version")):
			if i+1 < len(args) {
				version = args[i+1]
				skip = true
			}
			continue
		case skipArg(a):
			continue
		}
		if name == "" {
			name = a
		}
	}
	if name == "" {
		return nil
	}
	return &InstallRequest{Manager: ManagerDotnet, Packages: []Package{{Name: name, Version: version}}}
}

// parseGoArgs handles `go get pkg`, `go get pkg@v1`, multi-pkg variants. Bare
// `go get` / `go install` with no positional pkg → no request.
func parseGoArgs(args []string) *InstallRequest {
	var pkgs []Package
	for _, a := range args {
		if skipArg(a) {
			continue
		}
		if a == "." || a == "./..." {
			continue
		}
		idx := strings.LastIndex(a, "@")
		if idx <= 0 {
			pkgs = append(pkgs, Package{Name: a})
		} else {
			pkgs = append(pkgs, Package{Name: a[:idx], Version: a[idx+1:]})
		}
	}
	if len(pkgs) == 0 {
		return nil
	}
	return &InstallRequest{Manager: ManagerGo, Packages: pkgs}
}

// parseMavenArgs extracts the artifact spec from
// `mvn dependency:get -Dartifact=groupId:artifactId:version`.
func parseMavenArgs(args []string) *InstallRequest {
	for _, a := range args {
		if !strings.HasPrefix(a, "-Dartifact=") {
			continue
		}
		spec := strings.TrimPrefix(a, "-Dartifact=")
		parts := strings.Split(spec, ":")
		if len(parts) < 2 {
			continue
		}
		name := parts[0] + ":" + parts[1]
		ver := ""
		if len(parts) >= 3 {
			ver = parts[2]
		}
		return &InstallRequest{Manager: ManagerMaven, Packages: []Package{{Name: name, Version: ver}}}
	}
	return nil
}

// isFlag reports whether the token starts with '-' (treating '-' alone or
// e.g. `--no-progress` and `-D` as flags) but excluding "@scope/pkg" style.
func isFlag(t string) bool {
	return strings.HasPrefix(t, "-")
}

// skipArg reports whether the token should be ignored entirely when scanning
// for package names — flags and unresolved shell expansions.
func skipArg(t string) bool {
	return isFlag(t) || isShellExpansion(t)
}

// resolveRef returns an absolute (or working-directory-relative) path for a
// requirements file reference. Kept here so the Bash hook can hand it to the
// scanner directly.
func resolveRef(ref, workDir string) string {
	if filepath.IsAbs(ref) || workDir == "" {
		return ref
	}
	return filepath.Join(workDir, ref)
}
