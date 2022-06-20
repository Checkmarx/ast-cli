//go:build integration

package integration

import (
	"testing"
)

func TestPrDecorationMissingScanID(t *testing.T) {
	args := []string{
		"utils",
		"pr",
	}
	err, _ := executeCommand(t, args...)
	assertError(t, err, "failed creating PR Decoration: Please provide scan-id flag")
}

func TestPrDecorationMissingNamespaceFlag(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"--scan-id",
		"4e6dd120-c126-484e-91d9-161a8e4f2bb1",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "failed creating PR Decoration: Please provide namespace flag")
}

func TestPrDecorationMissingPRNumberFlag(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"--scan-id",
		"4e6dd120-c126-484e-91d9-161a8e4f2bb1",
		"--namespace",
		"jay-nanduri",
		"--repo-name",
		"testGHAction",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "failed creating PR Decoration: Please provide pr-number flag")
}

func TestPrDecorationMissingRepoFlag(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"--scan-id",
		"4e6dd120-c126-484e-91d9-161a8e4f2bb1",
		"--namespace",
		"jay-nanduri",
		"--pr-number",
		"1",
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "failed creating PR Decoration: Please provide repo-name flag")
}

func TestPRDecorationSuccessCase(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"--scan-id",
		"4e6dd120-c126-484e-91d9-161a8e4f2bb1",
		"--namespace",
		"jay-nanduri",
		"--pr-number",
		"1",
		"--repo-name",
		"testGHAction",
	}
	err, _ := executeCommand(t, args...)
	assertError(t, err, "Response status code 201")
}
