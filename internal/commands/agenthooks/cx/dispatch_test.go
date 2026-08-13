//go:build !integration

package cx

import (
	"os"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
)

func TestDispatchRoute_InvokesRegisteredHandler(t *testing.T) {
	agenthooks.ClearRoutes()
	t.Cleanup(agenthooks.ClearRoutes)

	called := false
	var argsDuring []string
	agenthooks.AddRoute("test-route", func() {
		called = true
		argsDuring = append([]string(nil), os.Args...)
	})

	origArgs := append([]string(nil), os.Args...)
	DispatchRoute("test-route")

	if !called {
		t.Fatal("expected registered handler to be called")
	}
	if len(argsDuring) != 2 {
		t.Fatalf("during dispatch os.Args len = %d, want 2; got %v", len(argsDuring), argsDuring)
	}
	if argsDuring[0] != origArgs[0] {
		t.Errorf("os.Args[0] during dispatch = %q, want %q", argsDuring[0], origArgs[0])
	}
	if argsDuring[1] != "test-route" {
		t.Errorf("os.Args[1] during dispatch = %q, want %q", argsDuring[1], "test-route")
	}
}

func TestDispatchRoute_RestoresOsArgs(t *testing.T) {
	agenthooks.ClearRoutes()
	t.Cleanup(agenthooks.ClearRoutes)

	agenthooks.AddRoute("claude-stop", func() {})

	prevArgs := append([]string(nil), os.Args...)
	t.Cleanup(func() { os.Args = prevArgs })

	orig := []string{"cx", "hooks", "claude-stop", "--extra"}
	os.Args = append([]string(nil), orig...)

	DispatchRoute("claude-stop")

	if len(os.Args) != len(orig) {
		t.Fatalf("os.Args not restored: got %v, want %v", os.Args, orig)
	}
	for i := range orig {
		if os.Args[i] != orig[i] {
			t.Fatalf("os.Args not restored: got %v, want %v", os.Args, orig)
		}
	}
}

func TestDispatchRoute_SelectsMatchingRouteOnly(t *testing.T) {
	agenthooks.ClearRoutes()
	t.Cleanup(agenthooks.ClearRoutes)

	var hit string
	agenthooks.AddRoute("route-a", func() { hit = "a" })
	agenthooks.AddRoute("route-b", func() { hit = "b" })

	DispatchRoute("route-b")
	if hit != "b" {
		t.Fatalf("hit = %q, want b", hit)
	}

	DispatchRoute("route-a")
	if hit != "a" {
		t.Fatalf("hit = %q, want a", hit)
	}
}

func TestDispatchRoute_ReplacesFullArgsSliceDuringDispatch(t *testing.T) {
	agenthooks.ClearRoutes()
	t.Cleanup(agenthooks.ClearRoutes)

	prevArgs := append([]string(nil), os.Args...)
	t.Cleanup(func() { os.Args = prevArgs })

	// Pretend cobra already parsed a longer argv; DispatchRoute must narrow it
	// to [prog, route] so agenthooks.Dispatch resolves the route from Args[1].
	orig := []string{"cx", "hooks", "cursor-stop", "ignored"}
	os.Args = append([]string(nil), orig...)

	var seen []string
	agenthooks.AddRoute("cursor-stop", func() {
		seen = append([]string(nil), os.Args...)
	})

	DispatchRoute("cursor-stop")

	if len(seen) != 2 || seen[1] != "cursor-stop" {
		t.Fatalf("during dispatch os.Args = %v, want [prog cursor-stop]", seen)
	}
	for i := range orig {
		if os.Args[i] != orig[i] {
			t.Fatalf("os.Args not restored after dispatch: got %v, want %v", os.Args, orig)
		}
	}
}
