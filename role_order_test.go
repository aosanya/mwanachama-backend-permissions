package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM14_OpenBug_ListRolesIgnoresTheDeclaredNewestFirstOrder(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	creationOrder := []string{"aaa_oldest", "mmm_middle", "zzz_newest"}
	for _, code := range creationOrder {
		if _, err := mgr.CreateRole(ctx, models.Role{Code: code, Name: code}); err != nil {
			t.Fatalf("CreateRole(%s): %v", code, err)
		}
	}

	out, err := mgr.ListRoles(ctx, models.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(out) != len(creationOrder) {
		t.Fatalf("ListRoles returned %d roles, want %d", len(out), len(creationOrder))
	}

	var got []string
	for _, r := range out {
		got = append(got, r.Code)
	}

	wantNewestFirst := []string{"zzz_newest", "mmm_middle", "aaa_oldest"}
	for i, code := range wantNewestFirst {
		if got[i] != code {
			t.Fatalf("PM14 open bug: permissions.operations.json's list_roles says %q, but ListRoles returned %v — "+
				"the newest-created role (zzz_newest) sorts last because the query orders by Code alphabetically, "+
				"not by creation time; want newest-first order %v",
				"List the bundles declared here, newest naming first.", got, wantNewestFirst)
		}
	}
}
