package mwanachamapermissions_test

import (
	"context"
	"errors"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestPM6_OpenBug_CreateGrantAcceptsARetiredRoleCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, permissions.Role{Code: "viewer", Name: "Viewer"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.AddAction(ctx, "viewer", "agency.module.list"); err != nil {
		t.Fatalf("AddAction: %v", err)
	}
	if err := mgr.DeleteRole(ctx, "viewer"); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	_, err := mgr.CreateGrant(ctx, permissions.Grant{
		Subject: "mallory", Scope: "instance:x", RoleCode: "viewer",
	})
	if !errors.Is(err, permissions.ErrRoleNotFound) {
		t.Fatalf("PM6: CreateGrant against a retired role returned err=%v, want ErrRoleNotFound — "+
			"a caller gets a 201 for a grant MayPerform will silently never honour", err)
	}
}
