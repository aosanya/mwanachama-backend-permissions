package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestARetiredRoleCodeCannotBeRecreatedInAnySpellingToReviveItsGrants(t *testing.T) {
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
	if ok, err := pm.MayPerform(ctx, "member-a", "instance:alpha", "example.thing.update"); err != nil || !ok {
		t.Fatalf("precondition: the grant authorizes before retirement: ok=%v err=%v", ok, err)
	}
	if err := pm.DeleteRole(ctx, "editor"); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if ok, err := pm.MayPerform(ctx, "member-a", "instance:alpha", "example.thing.update"); err != nil || ok {
		t.Fatalf("precondition: retiring the role ends its grant: ok=%v err=%v", ok, err)
	}

	for _, code := range []string{"editor", "Editor", "EDITOR", "editor ", " editor", "editor\n"} {
		if r, err := pm.CreateRole(ctx, models.Role{Code: code, Name: "Editor again"}); err == nil {
			t.Errorf("a role was re-created under the retired code spelled %q (stored as %q)", code, r.Code)
			_, _ = pm.AddAction(ctx, code, "example.thing.update")
		}
		if ok, err := pm.MayPerform(ctx, "member-a", "instance:alpha", "example.thing.update"); err != nil || ok {
			t.Errorf("re-creating the retired role as %q revived member-a's grant: ok=%v err=%v", code, ok, err)
		}
	}
}
