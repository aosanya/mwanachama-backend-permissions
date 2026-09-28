package mwanachamapermissions_test

import (
	"context"
	"errors"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

const (
	alice = "0f3d1b8e-5a21-4c77-9f4e-2b6c0a11d3e9"
	bob   = "7c2a9f41-63be-4d0a-8e15-4f9ab2c77d10"
	shop  = "instance:6037d06c-753e-4747-bec8-14044a4903d3"
	other = "instance:11111111-2222-3333-4444-555555555555"
)

func seedViewer(t *testing.T, mgr permissions.PermissionsManager, ctx context.Context) {
	t.Helper()
	if _, err := mgr.CreateRole(ctx, permissions.Role{Code: "viewer", Name: "Viewer"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	for _, a := range []string{"agency.module.list", "agency.module.read"} {
		if _, err := mgr.AddAction(ctx, "viewer", a); err != nil {
			t.Fatalf("AddAction %s: %v", a, err)
		}
	}
}

func TestMayPerform_OnlyWhatTheRoleLists(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	if _, err := mgr.CreateGrant(ctx, permissions.Grant{
		Subject: alice, Scope: shop, RoleCode: "viewer",
	}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	for _, tc := range []struct {
		action string
		want   bool
	}{
		{"agency.module.list", true},
		{"agency.module.read", true},
		{"agency.goal.create", false},
		{"catalog.entry.set_visibility", false},
	} {
		got, err := mgr.MayPerform(ctx, alice, shop, tc.action)
		if err != nil {
			t.Fatalf("MayPerform(%s): %v", tc.action, err)
		}
		if got != tc.want {
			t.Errorf("MayPerform(%s) = %v, want %v", tc.action, got, tc.want)
		}
	}
}

func TestMayPerform_IsScoped(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	if _, err := mgr.CreateGrant(ctx, permissions.Grant{
		Subject: alice, Scope: shop, RoleCode: "viewer",
	}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	here, err := mgr.MayPerform(ctx, alice, shop, "agency.module.list")
	if err != nil || !here {
		t.Fatalf("expected the grant to hold in its own scope, got %v / %v", here, err)
	}
	elsewhere, err := mgr.MayPerform(ctx, alice, other, "agency.module.list")
	if err != nil {
		t.Fatalf("MayPerform(other): %v", err)
	}
	if elsewhere {
		t.Error("a grant in one scope must not reach another")
	}
	stranger, err := mgr.MayPerform(ctx, bob, shop, "agency.module.list")
	if err != nil {
		t.Fatalf("MayPerform(bob): %v", err)
	}
	if stranger {
		t.Error("a subject with no grant must be refused")
	}
}

func TestMayPerform_EverywhereScopeReachesEveryScope(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	if _, err := mgr.CreateGrant(ctx, permissions.Grant{
		Subject: alice, Scope: permissions.ScopeEverywhere, RoleCode: "viewer",
	}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	for _, scope := range []string{shop, other} {
		ok, err := mgr.MayPerform(ctx, alice, scope, "agency.module.list")
		if err != nil || !ok {
			t.Errorf("a grant made everywhere should hold in %s, got %v / %v", scope, ok, err)
		}
	}
}

func TestMayPerform_WithdrawnGrantPermitsNothing(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	if _, err := mgr.CreateGrant(ctx, permissions.Grant{
		Subject: alice, Scope: shop, RoleCode: "viewer",
	}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if err := mgr.WithdrawGrant(ctx, alice, shop, "viewer"); err != nil {
		t.Fatalf("WithdrawGrant: %v", err)
	}

	ok, err := mgr.MayPerform(ctx, alice, shop, "agency.module.list")
	if err != nil {
		t.Fatalf("MayPerform: %v", err)
	}
	if ok {
		t.Error("a withdrawn grant must permit nothing")
	}

	grants, err := mgr.ListGrants(ctx, permissions.GrantFilter{Subject: alice})
	if err != nil {
		t.Fatalf("ListGrants: %v", err)
	}
	if len(grants) != 0 {
		t.Errorf("a withdrawn grant should not be listed, got %+v", grants)
	}
}

func TestMayPerform_RetiredRolePermitsNothing(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	if _, err := mgr.CreateGrant(ctx, permissions.Grant{
		Subject: alice, Scope: shop, RoleCode: "viewer",
	}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if err := mgr.DeleteRole(ctx, "viewer"); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	ok, err := mgr.MayPerform(ctx, alice, shop, "agency.module.list")
	if err != nil {
		t.Fatalf("MayPerform: %v", err)
	}
	if ok {
		t.Error("retiring a role must stop every grant naming it")
	}
}

func TestMayPerform_RefusesAMalformedAction(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	for _, bad := range []string{"", "nodots", "Agency.Module.List", "agency..list", "agency.module.list; drop table grants"} {
		if _, err := mgr.MayPerform(ctx, alice, shop, bad); !errors.Is(err, permissions.ErrInvalidAction) {
			t.Errorf("MayPerform(%q) error = %v, want ErrInvalidAction", bad, err)
		}
	}
}

func TestActionsFor_IsTheUnionAcrossRoles(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	if _, err := mgr.CreateRole(ctx, permissions.Role{Code: "miner", Name: "Miner"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	for _, a := range []string{"catalog.suggestion.file", "agency.module.list"} {
		if _, err := mgr.AddAction(ctx, "miner", a); err != nil {
			t.Fatalf("AddAction: %v", err)
		}
	}
	for _, code := range []string{"viewer", "miner"} {
		if _, err := mgr.CreateGrant(ctx, permissions.Grant{
			Subject: alice, Scope: shop, RoleCode: code,
		}); err != nil {
			t.Fatalf("CreateGrant(%s): %v", code, err)
		}
	}

	got, err := mgr.ActionsFor(ctx, alice, shop)
	if err != nil {
		t.Fatalf("ActionsFor: %v", err)
	}
	want := []string{"agency.module.list", "agency.module.read", "catalog.suggestion.file"}
	if len(got) != len(want) {
		t.Fatalf("ActionsFor = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ActionsFor = %v, want %v", got, want)
		}
	}
}

func TestCreateGrant_RefusesAnUnknownRole(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, err := mgr.CreateGrant(ctx, permissions.Grant{
		Subject: alice, Scope: shop, RoleCode: "nobody",
	})
	if !errors.Is(err, permissions.ErrRoleNotFound) {
		t.Errorf("error = %v, want ErrRoleNotFound", err)
	}
}

func TestCreateGrant_RefusesAMalformedScope(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	for _, bad := range []string{"instance", "INSTANCE:abc", "instance:", "' or 1=1 --"} {
		_, err := mgr.CreateGrant(ctx, permissions.Grant{
			Subject: alice, Scope: bad, RoleCode: "viewer",
		})
		if !errors.Is(err, permissions.ErrInvalidScope) {
			t.Errorf("CreateGrant(scope %q) error = %v, want ErrInvalidScope", bad, err)
		}
	}
}

func TestCreateGrant_ReinstatesRatherThanDuplicating(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	g := permissions.Grant{Subject: alice, Scope: shop, RoleCode: "viewer"}
	if _, err := mgr.CreateGrant(ctx, g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if _, err := mgr.CreateGrant(ctx, g); !errors.Is(err, permissions.ErrDuplicateGrant) {
		t.Fatalf("a live grant made twice should refuse, got %v", err)
	}
	if err := mgr.WithdrawGrant(ctx, alice, shop, "viewer"); err != nil {
		t.Fatalf("WithdrawGrant: %v", err)
	}
	if _, err := mgr.CreateGrant(ctx, g); err != nil {
		t.Fatalf("re-granting after withdrawal should work, got %v", err)
	}

	grants, err := mgr.ListGrants(ctx, permissions.GrantFilter{Subject: alice})
	if err != nil {
		t.Fatalf("ListGrants: %v", err)
	}
	if len(grants) != 1 {
		t.Errorf("expected one row, not a second: %+v", grants)
	}
}

func TestSetActions_ReplacesTheWholeListAndMarksTheRoleSeeded(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	if _, err := mgr.SetActions(ctx, "viewer", []string{"agency.publication.list", "agency.module.list"}); err != nil {
		t.Fatalf("SetActions: %v", err)
	}
	actions, err := mgr.ListActions(ctx, "viewer")
	if err != nil {
		t.Fatalf("ListActions: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected the list replaced, got %+v", actions)
	}
	if actions[0].Action != "agency.module.list" || actions[1].Action != "agency.publication.list" {
		t.Errorf("actions = %+v", actions)
	}

	role, err := mgr.GetRole(ctx, "viewer")
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if !role.Seeded {
		t.Error("a seeded role should say so, so a hand edit can be refused")
	}
	if _, err := mgr.AddAction(ctx, "viewer", "agency.goal.create"); !errors.Is(err, permissions.ErrSeededRole) {
		t.Errorf("AddAction on a seeded role = %v, want ErrSeededRole", err)
	}
}

func TestSetActions_RefusesAMalformedActionBeforeWritingAny(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	_, err := mgr.SetActions(ctx, "viewer", []string{"agency.module.list", "NOPE"})
	if !errors.Is(err, permissions.ErrInvalidAction) {
		t.Fatalf("SetActions error = %v, want ErrInvalidAction", err)
	}
	actions, err := mgr.ListActions(ctx, "viewer")
	if err != nil {
		t.Fatalf("ListActions: %v", err)
	}
	if len(actions) != 2 {
		t.Errorf("a refused SetActions must leave the old list alone, got %+v", actions)
	}
}

func TestActionsAnywhere_IsTheUnionAcrossEveryScope(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedViewer(t, mgr, ctx)

	if _, err := mgr.CreateRole(ctx, permissions.Role{Code: "miner", Name: "Miner"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.AddAction(ctx, "miner", "catalog.suggestion.file"); err != nil {
		t.Fatalf("AddAction: %v", err)
	}
	if _, err := mgr.CreateGrant(ctx, permissions.Grant{Subject: alice, Scope: shop, RoleCode: "viewer"}); err != nil {
		t.Fatalf("CreateGrant(shop): %v", err)
	}
	if _, err := mgr.CreateGrant(ctx, permissions.Grant{Subject: alice, Scope: other, RoleCode: "miner"}); err != nil {
		t.Fatalf("CreateGrant(other): %v", err)
	}

	here, err := mgr.ActionsFor(ctx, alice, shop)
	if err != nil {
		t.Fatalf("ActionsFor: %v", err)
	}
	if len(here) != 2 {
		t.Errorf("ActionsFor(shop) = %v, want only the viewer's two", here)
	}

	anywhere, err := mgr.ActionsAnywhere(ctx, alice)
	if err != nil {
		t.Fatalf("ActionsAnywhere: %v", err)
	}
	if len(anywhere) != 3 {
		t.Fatalf("ActionsAnywhere = %v, want the union of both scopes", anywhere)
	}

	if err := mgr.WithdrawGrant(ctx, alice, other, "miner"); err != nil {
		t.Fatalf("WithdrawGrant: %v", err)
	}
	after, err := mgr.ActionsAnywhere(ctx, alice)
	if err != nil {
		t.Fatalf("ActionsAnywhere: %v", err)
	}
	if len(after) != 2 {
		t.Errorf("a withdrawn grant must leave the union, got %v", after)
	}
}

func TestPM6_OpenBug_DeleteRoleLeavesTheCodePermanentlyTaken(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, permissions.Role{Code: "temp", Name: "Temp"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if err := mgr.DeleteRole(ctx, "temp"); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	roles, err := mgr.ListRoles(ctx, permissions.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	for _, r := range roles {
		if r.Code == "temp" {
			t.Fatalf("a retired role must not appear in ListRoles, got %+v", r)
		}
	}

	if _, err := mgr.CreateRole(ctx, permissions.Role{Code: "temp", Name: "Temp Again"}); err != nil {
		t.Fatalf("PM6 open bug: CreateRole(%q) after DeleteRole got %v, want success — "+
			"ListRoles already treats a retired role's code as free, so CreateRole must too, "+
			"not permanently lock it out with ErrDuplicateRole", "temp", err)
	}
}
