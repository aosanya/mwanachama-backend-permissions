package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM10_OpenBug_CreateGrantAcceptsARetiredRole(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, models.Role{Code: "writer", Name: "Writer"}); err != nil {
		t.Fatalf("create role: %v", err)
	}
	if _, err := mgr.AddAction(ctx, "writer", "catalog.entry.upsert"); err != nil {
		t.Fatalf("add action: %v", err)
	}
	if err := mgr.DeleteRole(ctx, "writer"); err != nil {
		t.Fatalf("retire role: %v", err)
	}

	g, err := mgr.CreateGrant(ctx, models.Grant{Subject: "bob", Scope: "instance:abc", RoleCode: "writer"})
	if err == nil {
		t.Fatalf("PM10 open bug: CreateGrant(subject=bob, role=writer) after DeleteRole(writer) returned %+v with no error, want it refused (ErrRoleNotFound or equivalent) the same way CreateGrant already refuses a role code that was never created; instead it stored a grant that MayPerform will silently deny forever, with no signal to the caller at write time", g)
	}
}
