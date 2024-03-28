//go:build integration

package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

func TestCreateScanAndProjectWithGroupFFTrue(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: "ACCESS_MANAGEMENT_ENABLED", Status: true}}
	execCmdNilAssertion(
		t,
		"scan", "create", "--project-name", "new-project", "-b", "dummy_branch", "-s", ".", "--project-groups", "group",
	)
}

func TestCreateScanAndProjectWithGroupFFFalse(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: "ACCESS_MANAGEMENT_ENABLED", Status: false}}
	execCmdNilAssertion(
		t,
		"scan", "create", "--project-name", "new-project", "-b", "dummy_branch", "-s", ".", "--project-groups", "group",
	)
}
func TestCreateProjectWithGroupFFTrue(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: "ACCESS_MANAGEMENT_ENABLED", Status: true}}
	execCmdNilAssertion(
		t, "project", "create", "--project-name", "new-project", "--groups", "group",
	)
}

func TestCreateProjectWithGroupFFFalse(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: "ACCESS_MANAGEMENT_ENABLED", Status: false}}
	execCmdNilAssertion(
		t,
		"project", "create", "--project-name", "new-project", "--groups", "group",
	)
}

func TestCreateScanForExistingProjectWithGroupFFTrue(t *testing.T) {
	mock.Flags = wrappers.FeatureFlagsResponseModel{{Name: "ACCESS_MANAGEMENT_ENABLED", Status: true}}
	execCmdNilAssertion(
		t,
		"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch", "-s", ".", "--project-groups", "group",
	)
}
