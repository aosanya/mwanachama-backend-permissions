package mwanachamapermissions_test

import (
	"context"
	"errors"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestPM18_DeleteRoleRefusesASeededRoleAndLeavesItLive(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, permissions.Role{Code: "writer", Name: "Writer"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.SeedActions(ctx, "writer", []string{"catalog.entry.upsert"}); err != nil {
		t.Fatalf("SeedActions: %v", err)
	}

	if err := mgr.DeleteRole(ctx, "writer"); !errors.Is(err, permissions.ErrSeededRole) {
		t.Fatalf("DeleteRole on a seeded role = %v, want ErrSeededRole", err)
	}

	role, err := mgr.GetRole(ctx, "writer")
	if err != nil {
		t.Fatalf("GetRole after the refused delete: %v", err)
	}
	if role.Deleted {
		t.Fatalf("the refused delete still retired the role: %+v", role)
	}
}
