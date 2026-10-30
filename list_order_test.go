package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM15_ListRolesAnswersNewestFirst(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, models.Role{Code: "a_older", Name: "Older"}); err != nil {
		t.Fatalf("create older role: %v", err)
	}
	if _, err := mgr.CreateRole(ctx, models.Role{Code: "b_newer", Name: "Newer"}); err != nil {
		t.Fatalf("create newer role: %v", err)
	}

	got, err := mgr.ListRoles(ctx, models.RoleFilter{})
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(got))
	}

	if got[0].Code != "b_newer" || got[1].Code != "a_older" {
		t.Fatalf("PM15 open bug: permissions.operations.json's list_roles description promises %q but ListRoles(RoleFilter{}) returned %q then %q — the role created first comes first, not last: a caller relying on the declared order sees the opposite of what was declared",
			"List the bundles declared here, newest naming first.", got[0].Code, got[1].Code)
	}
}
