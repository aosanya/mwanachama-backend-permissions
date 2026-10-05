package mwanachamapermissions_test

import (
	"context"
	"testing"
	"time"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM15_OpenBug_ListRolesIgnoresItsOwnDeclaredNewestFirstOrder(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, models.Role{Code: "aaa_old", Name: "Declared first"}); err != nil {
		t.Fatalf("create aaa_old: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := mgr.CreateRole(ctx, models.Role{Code: "bbb_new", Name: "Declared last"}); err != nil {
		t.Fatalf("create bbb_new: %v", err)
	}

	roles, err := mgr.ListRoles(ctx, models.RoleFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("want 2 roles, got %d", len(roles))
	}
	if roles[0].Code != "bbb_new" || roles[1].Code != "aaa_old" {
		t.Fatalf("PM15: permissions.operations.json's list_roles declares %q; "+
			"bbb_new was declared after aaa_old and should lead the list; got order %q, %q",
			"List the bundles declared here, newest naming first.", roles[0].Code, roles[1].Code)
	}
}
