package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM14_OpenBug_ListRolesIgnoresDeclaredNewestFirstOrder(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	created := []string{"role_alpha", "role_mid", "role_zeta"}
	for _, code := range created {
		if _, err := mgr.CreateRole(ctx, models.Role{Code: code, Name: code}); err != nil {
			t.Fatalf("create %s: %v", code, err)
		}
	}

	got, err := mgr.ListRoles(ctx, models.RoleFilter{})
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	var order []string
	for _, r := range got {
		order = append(order, r.Code)
	}

	want := []string{"role_zeta", "role_mid", "role_alpha"}
	ok := len(order) == len(want)
	for i := range want {
		if !ok || order[i] != want[i] {
			ok = false
			break
		}
	}
	if !ok {
		t.Fatalf("PM14 open bug: created roles in order %v (role_zeta declared most recently); ListRoles should answer newest-declared-first (%v) per permissions.operations.json's list_roles description, but returned %v — role.go's ListRoles calls Order(columnName(\"Code\")) with no reference to CreatedAt at all, so the order a caller sees depends on what the codes happen to sort to, never on which role was declared most recently", created, want, order)
	}
}
