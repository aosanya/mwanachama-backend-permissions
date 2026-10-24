package mwanachamapermissions_test

import (
	"context"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestPM18_OpenBug_DeleteRoleDestroysASeededRoleForever(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, permissions.Role{Code: "writer", Name: "Writer"}); err != nil {
		t.Fatalf("precondition: CreateRole: %v", err)
	}
	if _, err := mgr.SeedActions(ctx, "writer", []string{"catalog.entry.upsert"}); err != nil {
		t.Fatalf("precondition: SeedActions: %v", err)
	}
	role, err := mgr.GetRole(ctx, "writer")
	if err != nil {
		t.Fatalf("precondition: GetRole: %v", err)
	}
	if !role.Seeded {
		t.Fatalf("precondition: role is not Seeded after SetActions, got %+v", role)
	}
	if _, err := mgr.AddAction(ctx, "writer", "permissions.grant.create"); err == nil {
		t.Fatalf("precondition: AddAction on a seeded role must be refused, got nil error")
	}

	deleteErr := mgr.DeleteRole(ctx, "writer")
	if deleteErr == nil {
		t.Errorf("PM18 open bug: DeleteRole(\"writer\") on a seeded role answered nil (success); want it refused the same way AddAction/RemoveAction already refuse any hand write to a role the seeding pass owns")
	}

	role, err = mgr.GetRole(ctx, "writer")
	if err != nil {
		t.Fatalf("GetRole after delete: %v", err)
	}
	if !role.Deleted {
		t.Fatalf("precondition: DeleteRole did not retire the role, got %+v", role)
	}

	_, recreateErr := mgr.CreateRole(ctx, permissions.Role{Code: "writer", Name: "Writer"})
	if recreateErr == nil {
		t.Fatalf("precondition: expected the retired code to stay taken (this module's own documented behaviour)")
	}
	t.Errorf("PM18 open bug: a caller holding only permissions.role.delete permanently destroyed role %q, which the seeding pass marked Seeded=true and which AddAction/RemoveAction already refuse to touch (DeleteRole err=%v); CreateRole(%q) now answers %v forever, so the seeding pass itself can never reconstitute the role it declared under this code again", "writer", deleteErr, "writer", recreateErr)
}
