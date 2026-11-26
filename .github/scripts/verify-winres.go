//go:build ignore

// verify-winres checks Windows exes for the Checkmarx version resource: go run .github/scripts/verify-winres.go <os> <X.Y.Z> <cx.exe>...
package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
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
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: verify-winres <os> <X.Y.Z> <exe>...")
		os.Exit(2)
	}
	targetOS, version := os.Args[1], os.Args[2]
	// the GoReleaser build hook runs for every target, only Windows executables carry a version resource
	if targetOS != "windows" {
		fmt.Printf("target os %q is not windows, skipping\n", targetOS)
		return
	}
	if !versionRe.MatchString(version) || version == "0.0.0" {
		fmt.Fprintf(os.Stderr, "invalid expected version %q (want X.Y.Z, not the 0.0.0 placeholder)\n", version)
		os.Exit(1)
	}
	expected := []string{"Checkmarx", "Checkmarx One CLI", "Copyright (c) Checkmarx Ltd.", version}
	failed := false
	for _, exe := range os.Args[3:] {
		data, err := os.ReadFile(exe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", exe, err)
			os.Exit(1)
		}
		exeFailed := false
		for _, s := range expected {
			needle := append(append([]byte{0, 0}, utf16le(s)...), 0, 0)
			if !bytes.Contains(data, needle) {
				fmt.Fprintf(os.Stderr, "%s is missing version info %q\n", exe, s)
				exeFailed = true
			}
		}
		if exeFailed {
			failed = true
		} else {
			fmt.Printf("%s OK (version %s)\n", exe, version)
		}
	}
	if failed {
		os.Exit(1)
	}
}
