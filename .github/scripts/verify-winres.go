//go:build ignore

// verify-winres checks Windows exes for the Checkmarx version resource: go run .github/scripts/verify-winres.go <X.Y.Z> <cx.exe>...
package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf16"
)

var versionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func utf16le(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = append(b, byte(u), byte(u>>8))
	}
	return b
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: verify-winres <X.Y.Z> <exe>...")
		os.Exit(2)
	}
	version := os.Args[1]
	if !versionRe.MatchString(version) || version == "0.0.0" {
		fmt.Fprintf(os.Stderr, "invalid expected version %q (want X.Y.Z, not the 0.0.0 placeholder)\n", version)
		os.Exit(1)
	}
	expected := []string{"Checkmarx", "Checkmarx One CLI", "Copyright (c) Checkmarx Ltd.", version}
	failed := false
	for _, exe := range os.Args[2:] {
		// the GoReleaser build hook runs for every target; only Windows executables carry a version resource
		if !strings.HasSuffix(strings.ToLower(exe), ".exe") {
			fmt.Printf("%s is not a Windows executable, skipping\n", exe)
			continue
		}
		data, err := os.ReadFile(exe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", exe, err)
			os.Exit(1)
		}
		for _, s := range expected {
			needle := append(append([]byte{0, 0}, utf16le(s)...), 0, 0)
			if !bytes.Contains(data, needle) {
				fmt.Fprintf(os.Stderr, "%s is missing version info %q\n", exe, s)
				failed = true
			}
		}
		if !failed {
			fmt.Printf("%s OK (version %s)\n", exe, version)
		}
	}
	if failed {
		os.Exit(1)
	}
}
