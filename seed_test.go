package mwanachamapermissions_test

import (
	"context"
	"errors"
	"os"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestDeclaredActions_ReadsEveryActionAndDeduplicates(t *testing.T) {
	got, err := permissions.DeclaredActions(permissions.Operations(), permissions.Operations())
	if err != nil {
		t.Fatalf("DeclaredActions: %v", err)
	}
	if len(got) != 14 {
		t.Fatalf("got %d actions, want this module's 14 declared once: %v", len(got), got)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Fatalf("actions are not sorted and unique: %v", got)
		}
	}
}

func TestDeclaredActions_ReadsTheRealModulesBesideIt(t *testing.T) {
	for _, path := range []string{
		"../mwanachama-backend-catalog/catalog.operations.json",
		"../mwanachama-backend-agency/agency.operations.json",
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Skipf("%s is not here: %v", path, err)
		}
		got, err := permissions.DeclaredActions(raw)
		if err != nil {
			t.Fatalf("DeclaredActions(%s): %v", path, err)
		}
		if len(got) == 0 {
			t.Fatalf("%s declared no actions", path)
		}
	}
}

func TestDeclaredActions_RefusesAnOperationWithNoAction(t *testing.T) {
	_, err := permissions.DeclaredActions([]byte(`{"operations":{"x":{"method":"GET","path":"/x","call":"X","returns":[{"body":true}]}}}`))
	if err == nil {
		t.Fatal("an operation naming no action must be refused, not skipped")
	}
}

func TestSeed_CreatesTheRoleAndOwnsItsActions(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	written, err := mgr.Seed(ctx, "owner", "Owner", permissions.Operations())
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if len(written) != 14 {
		t.Fatalf("seeded %d actions, want 14", len(written))
	}

	role, err := mgr.GetRole(ctx, "owner")
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if !role.Seeded {
		t.Error("a seeded role must say so, so a hand edit is refused")
	}
	if _, err := mgr.AddAction(ctx, "owner", "x.y.z"); !errors.Is(err, permissions.ErrSeededRole) {
		t.Errorf("AddAction on a seeded role = %v, want ErrSeededRole", err)
	}
}

func TestSeed_IsIdempotentAndDropsWhatAModuleStoppedDeclaring(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.Seed(ctx, "owner", "Owner", permissions.Operations()); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	first, err := mgr.ListActions(ctx, "owner")
	if err != nil {
		t.Fatalf("ListActions: %v", err)
	}

	if _, err := mgr.Seed(ctx, "owner", "Owner", permissions.Operations()); err != nil {
		t.Fatalf("Seed again: %v", err)
	}
	second, err := mgr.ListActions(ctx, "owner")
	if err != nil {
		t.Fatalf("ListActions: %v", err)
	}
	if len(first) != len(second) {
		t.Errorf("seeding twice changed the count: %d then %d", len(first), len(second))
	}

	narrower := []byte(`{"operations":{"only":{"method":"GET","path":"/only","call":"Only","action":"permissions.role.list","returns":[{"body":true}]}}}`)
	if _, err := mgr.Seed(ctx, "owner", "Owner", narrower); err != nil {
		t.Fatalf("Seed narrower: %v", err)
	}
	third, err := mgr.ListActions(ctx, "owner")
	if err != nil {
		t.Fatalf("ListActions: %v", err)
	}
	if len(third) != 1 || third[0].Action != "permissions.role.list" {
		t.Errorf("a re-seed must drop what is no longer declared, got %+v", third)
	}
}

func TestSeed_TheSeededOwnerCanActuallyDoEverything(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.Seed(ctx, "owner", "Owner", permissions.Operations()); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if _, err := mgr.CreateGrant(ctx, permissions.Grant{
		Subject: alice, Scope: shop, RoleCode: "owner",
	}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	declared, err := permissions.DeclaredActions(permissions.Operations())
	if err != nil {
		t.Fatalf("DeclaredActions: %v", err)
	}
	for _, a := range declared {
		ok, err := mgr.MayPerform(ctx, alice, shop, a)
		if err != nil {
			t.Fatalf("MayPerform(%s): %v", a, err)
		}
		if !ok {
			t.Errorf("a seeded owner should be permitted %s", a)
		}
	}
	ok, err := mgr.MayPerform(ctx, alice, shop, "catalog.entry.upsert")
	if err != nil {
		t.Fatalf("MayPerform: %v", err)
	}
	if ok {
		t.Error("seeding from one module's operations must not permit another's")
	}
}
