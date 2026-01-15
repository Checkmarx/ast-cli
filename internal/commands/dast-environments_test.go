//go:build !integration

package commands

import (
	"testing"
)

func TestDastEnvironmentsHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "dast-environments")
}

func TestDastEnvironmentsNoSub(t *testing.T) {
	execCmdNilAssertion(t, "dast-environments")
}

func TestDastEnvironmentsList(t *testing.T) {
	execCmdNilAssertion(t, "dast-environments", "list")
}

func TestDastEnvironmentsListWithFormat(t *testing.T) {
	execCmdNilAssertion(t, "dast-environments", "list", "--format", "json")
}

func TestDastEnvironmentsListWithFilters(t *testing.T) {
	execCmdNilAssertion(t, "dast-environments", "list", "--filter", "from=1,to=10")
}

func TestDastEnvironmentsListWithSearch(t *testing.T) {
	execCmdNilAssertion(t, "dast-environments", "list", "--filter", "search=test")
}

func TestDastEnvironmentsListWithSort(t *testing.T) {
	execCmdNilAssertion(t, "dast-environments", "list", "--filter", "sort=domain:asc")
}
