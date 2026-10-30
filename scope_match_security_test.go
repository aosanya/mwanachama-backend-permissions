package mwanachamapermissions_test

import (
	"context"
	"errors"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestAGrantAuthorizesOnlyItsExactScopeAndAction(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)

	if _, err := pm.CreateRole(ctx, models.Role{Code: "editor", Name: "Editor"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := pm.AddAction(ctx, "editor", "example.thing.update"); err != nil {
		t.Fatalf("AddAction: %v", err)
	}
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "member-a", Scope: "instance:alpha", RoleCode: "editor", GrantedBy: "admin"}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	ok, err := pm.MayPerform(ctx, "member-a", "instance:alpha", "example.thing.update")
	if err != nil || !ok {
		t.Fatalf("precondition: the grant authorizes its own scope and action: ok=%v err=%v", ok, err)
	}

	for _, scope := range []string{"instance:alphabet", "instance:alph", "instance:ALPHA", "instance:alpha/", "instance:alpha:x", "instance:%", "instance:_lpha", "*"} {
		ok, err := pm.MayPerform(ctx, "member-a", scope, "example.thing.update")
		if err != nil || ok {
			t.Errorf("a grant on instance:alpha authorized scope %q: ok=%v err=%v", scope, ok, err)
		}
	}

	for _, subject := range []string{"member-A", "member-", "member-a ", "%", "member_a"} {
		ok, err := pm.MayPerform(ctx, subject, "instance:alpha", "example.thing.update")
		if err != nil || ok {
			t.Errorf("a grant to member-a authorized subject %q: ok=%v err=%v", subject, ok, err)
		}
	}

	for _, action := range []string{"example.thing.updat", "example.thing.update.more", "example.thing.%", "example.thing.updat_"} {
		ok, _ := pm.MayPerform(ctx, "member-a", "instance:alpha", action)
		if ok {
			t.Errorf("a grant of example.thing.update authorized action %q", action)
		}
	}
	if _, err := pm.MayPerform(ctx, "member-a", "instance:alpha", "Example.Thing.Update"); !errors.Is(err, permissions.ErrInvalidAction) {
		t.Errorf("a case-variant action was not refused as invalid: %v", err)
	}
}

func TestARetiredRolesCodeCannotBeRecreatedToReviveItsGrants(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)

	if _, err := pm.CreateRole(ctx, models.Role{Code: "editor", Name: "Editor"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "member-a", Scope: "instance:alpha", RoleCode: "editor", GrantedBy: "admin"}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if err := pm.DeleteRole(ctx, "editor"); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	if _, err := pm.CreateRole(ctx, models.Role{Code: "editor", Name: "Editor again"}); !errors.Is(err, permissions.ErrDuplicateRole) {
		t.Fatalf("a retired role's code was created again: err=%v", err)
	}
	if _, err := pm.UpdateRole(ctx, models.Role{Code: "editor", Name: "Renamed", Deleted: false}); err != nil {
		t.Fatalf("UpdateRole on the retired role: %v", err)
	}
	if _, err := pm.AddAction(ctx, "editor", "example.thing.update"); err != nil && !errors.Is(err, permissions.ErrRoleRetired) {
		t.Fatalf("AddAction on the retired role: %v", err)
	}

	ok, err := pm.MayPerform(ctx, "member-a", "instance:alpha", "example.thing.update")
	if err != nil || ok {
		t.Errorf("a standing grant on a retired role authorized after a rename and an added action: ok=%v err=%v", ok, err)
	}
}
