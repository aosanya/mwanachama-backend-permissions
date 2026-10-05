package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM15_OpenBug_ListRolesIgnoresTheDeclaredNewestFirstOrder(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	for _, code := range []string{"role_c", "role_a", "role_b"} {
		if _, err := mgr.CreateRole(ctx, models.Role{Code: code, Name: code}); err != nil {
			t.Fatalf("create %s: %v", code, err)
		}
	}

	got, err := mgr.ListRoles(ctx, models.RoleFilter{})
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("list roles: got %d roles, want 3", len(got))
	}

	gotOrder := []string{got[0].Code, got[1].Code, got[2].Code}
	wantNewestFirst := []string{"role_b", "role_a", "role_c"}
	for i := range wantNewestFirst {
		if gotOrder[i] != wantNewestFirst[i] {
			t.Fatalf("PM15 open bug: ListRoles(RoleFilter{}) returned %v in creation order role_c,role_a,role_b; want %v (newest-named first, per permissions.operations.json's list_roles description), because the implementation orders by Code ascending instead of by creation time", gotOrder, wantNewestFirst)
		}
	}
}
