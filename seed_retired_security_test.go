package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

var seedRetiredOperations = []byte(`{"operations":{
	"read_thing":{"method":"GET","path":"/things","call":"ListThings","action":"example.thing.list","returns":[{"body":true}]},
	"make_thing":{"method":"POST","path":"/things","call":"CreateThing","action":"example.thing.create","returns":[{"body":true}]}
}}`)

func TestReseedingARetiredRoleDoesNotReinstateItsGrants(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)

	if _, err := pm.CreateRole(ctx, models.Role{Code: "studio_operator", Name: "Studio Operator"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := pm.AddAction(ctx, "studio_operator", "example.thing.list"); err != nil {
		t.Fatalf("AddAction: %v", err)
	}
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "member-a", Scope: models.ScopeEverywhere, RoleCode: "studio_operator", GrantedBy: "admin"}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if ok, err := pm.MayPerform(ctx, "member-a", "instance:one", "example.thing.list"); err != nil || !ok {
		t.Fatalf("precondition: a hand-made role's grant should permit its action, got %v, %v", ok, err)
	}

	if err := pm.DeleteRole(ctx, "studio_operator"); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if ok, _ := pm.MayPerform(ctx, "member-a", "instance:one", "example.thing.list"); ok {
		t.Fatalf("precondition: a retired role's grant should permit nothing")
	}

	if _, err := pm.Seed(ctx, "studio_operator", "Studio Operator", seedRetiredOperations); err != nil {
		t.Fatalf("re-Seed: %v", err)
	}

	role, err := pm.GetRole(ctx, "studio_operator")
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if !role.Deleted {
		t.Errorf("re-seeding un-retired the role: deleted = false, want true")
	}
	for _, action := range []string{"example.thing.list", "example.thing.create"} {
		if ok, _ := pm.MayPerform(ctx, "member-a", "instance:one", action); ok {
			t.Errorf("re-seeding a retired role let its old grant perform %q again", action)
		}
	}
	if held, _ := pm.ActionsAnywhere(ctx, "member-a"); len(held) != 0 {
		t.Errorf("re-seeding a retired role left the subject holding %v, want nothing", held)
	}
}
