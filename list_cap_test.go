package mwanachamapermissions_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM8_OpenBug_ListRolesHasNoDefaultCap(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	const seeded = 150
	for i := 0; i < seeded; i++ {
		if _, err := mgr.CreateRole(ctx, models.Role{Code: fmt.Sprintf("role_%03d", i), Name: fmt.Sprintf("Role %03d", i)}); err != nil {
			t.Fatalf("create role %d: %v", i, err)
		}
	}

	got, err := mgr.ListRoles(ctx, models.RoleFilter{})
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(got) >= seeded {
		t.Fatalf("PM8 open bug: ListRoles(RoleFilter{}) with no limit returned all %d seeded roles; a caller who names no limit should get a capped default page, not the whole live table", len(got))
	}
}

func TestPM8_OpenBug_ListGrantsHasNoDefaultCap(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, models.Role{Code: "writer", Name: "Writer"}); err != nil {
		t.Fatalf("create role: %v", err)
	}

	const seeded = 150
	for i := 0; i < seeded; i++ {
		subject := fmt.Sprintf("account:%032d", i)
		if _, err := mgr.CreateGrant(ctx, models.Grant{Subject: subject, Scope: models.ScopeEverywhere, RoleCode: "writer"}); err != nil {
			t.Fatalf("create grant %d: %v", i, err)
		}
	}

	got, err := mgr.ListGrants(ctx, models.GrantFilter{})
	if err != nil {
		t.Fatalf("list grants: %v", err)
	}
	if len(got) >= seeded {
		t.Fatalf("PM8 open bug: ListGrants(GrantFilter{}) with no limit returned all %d seeded grants; a caller who names no limit should get a capped default page, not the whole live table", len(got))
	}
}
