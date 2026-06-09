package cx

import (
	"os"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
)

// DispatchRoute calls the handler registered under route without requiring the
// caller to mutate os.Args. Use this when the route name is already known (e.g.
// from cmd.Use after Cobra has consumed it from os.Args).
func DispatchRoute(route string) {
	saved := os.Args
	os.Args = []string{saved[0], route}
	agenthooks.Dispatch()
	os.Args = saved
}
