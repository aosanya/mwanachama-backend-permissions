package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM10_OpenBug_AddActionAcceptsARetiredRole(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, models.Role{Code: "writer", Name: "Writer"}); err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := mgr.DeleteRole(ctx, "writer"); err != nil {
		t.Fatalf("retire role: %v", err)
	}

	ra, err := mgr.AddAction(ctx, "writer", "catalog.entry.upsert")
	if err == nil {
		t.Fatalf("PM10 open bug: AddAction(role=writer, action=catalog.entry.upsert) after DeleteRole(writer) returned %+v with no error, want it refused (ErrRoleNotFound or equivalent) the same way CreateGrant should refuse a retired role code; instead it stored a RoleAction naming a role that can never permit anything again, with a 201-shaped success that tells the caller the action is now granted when it never will be", ra)
	}
}
