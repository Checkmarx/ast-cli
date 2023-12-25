package commands

import (
	"testing"
)

func TestChatHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "chat")
}
