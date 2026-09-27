package routes_test

import (
	"sort"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

func TestBuild_EveryDeclaredOperationMatchesTheManager(t *testing.T) {
	var nilManager permissions.PermissionsManager = (*stubManager)(nil)
	built, err := routes.Build(nilManager)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(built) == 0 {
		t.Fatal("no routes were generated")
	}

	seen := map[string]bool{}
	for _, r := range built {
		if r.Action == "" {
			t.Errorf("route %s %s carries no action, so nothing can gate it", r.Method, r.Path)
		}
		key := r.Method + " " + r.Path
		if seen[key] {
			t.Errorf("two operations claim %s", key)
		}
		seen[key] = true
	}
}

func TestNothingIsAnonymous(t *testing.T) {
	if len(routes.AnonymousActions) != 0 {
		t.Errorf("permissions has no public surface; AnonymousActions = %v", routes.AnonymousActions)
	}
}

func TestEveryActionIsWellFormedAndNamespaced(t *testing.T) {
	built, err := routes.Build((*stubManager)(nil))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var actions []string
	for _, r := range built {
		actions = append(actions, r.Action)
	}
	sort.Strings(actions)
	for _, a := range actions {
		if len(a) < len("permissions.") || a[:len("permissions.")] != "permissions." {
			t.Errorf("action %q is not namespaced to this module", a)
		}
	}
}
