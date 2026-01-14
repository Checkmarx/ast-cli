//go:build !integration

package commands

import (
	"testing"
)

func TestEnvironmentsHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "environments")
}

func TestEnvironmentsNoSub(t *testing.T) {
	execCmdNilAssertion(t, "environments")
}

func TestEnvironmentsList(t *testing.T) {
	execCmdNilAssertion(t, "environments", "list")
}

func TestEnvironmentsListWithFormat(t *testing.T) {
	execCmdNilAssertion(t, "environments", "list", "--format", "json")
}

func TestEnvironmentsListWithFilters(t *testing.T) {
	execCmdNilAssertion(t, "environments", "list", "--filter", "from=1,to=10")
}

func TestEnvironmentsListWithSearch(t *testing.T) {
	execCmdNilAssertion(t, "environments", "list", "--filter", "search=test")
}

func TestEnvironmentsListWithSort(t *testing.T) {
	execCmdNilAssertion(t, "environments", "list", "--filter", "sort=domain:asc")
}
