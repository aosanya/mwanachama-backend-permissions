package mwanachamapermissions_test

import (
	"context"
	"slices"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestTheActionListingAndTheGateAgreeThroughEveryLifecycleStep(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)

	subjects := []string{"member-a", "member-b"}
	scopes := []string{"instance:alpha", "instance:beta", "*"}
	actions := []string{"example.thing.read", "example.thing.update", "example.thing.delete", "example.other.read"}

	agree := func(step string) {
		t.Helper()
		for _, subject := range subjects {
			anywhere, err := pm.ActionsAnywhere(ctx, subject)
			if err != nil {
				t.Fatalf("%s: ActionsAnywhere(%s): %v", step, subject, err)
			}
			for _, scope := range scopes {
				listed, err := pm.ActionsFor(ctx, subject, scope)
				if err != nil {
					t.Fatalf("%s: ActionsFor(%s, %s): %v", step, subject, scope, err)
				}
				for _, action := range actions {
					allowed, err := pm.MayPerform(ctx, subject, scope, action)
					if err != nil {
						t.Fatalf("%s: MayPerform(%s, %s, %s): %v", step, subject, scope, action, err)
					}
					if allowed != slices.Contains(listed, action) {
						t.Errorf("%s: %s in %s, action %s: the gate answered %v but ActionsFor listed %v — a review reading the listing would see a different authority than the gate enforces", step, subject, scope, action, allowed, listed)
					}
					if allowed && !slices.Contains(anywhere, action) {
						t.Errorf("%s: %s may perform %s in %s but ActionsAnywhere listed only %v", step, subject, action, scope, anywhere)
					}
				}
			}
			for _, action := range anywhere {
				held := false
				for _, scope := range scopes {
					if ok, _ := pm.MayPerform(ctx, subject, scope, action); ok {
						held = true
					}
				}
				if !held {
					t.Errorf("%s: ActionsAnywhere listed %s for %s, which the gate refuses in every scope asked", step, action, subject)
				}
			}
		}
	}

	must := func(err error, what string) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}

	_, err := pm.CreateRole(ctx, models.Role{Code: "editor", Name: "Editor"})
	must(err, "CreateRole editor")
	_, err = pm.CreateRole(ctx, models.Role{Code: "viewer", Name: "Viewer"})
	must(err, "CreateRole viewer")
	for _, action := range []string{"example.thing.read", "example.thing.update", "example.thing.delete"} {
		_, err = pm.AddAction(ctx, "editor", action)
		must(err, "AddAction editor "+action)
	}
	_, err = pm.AddAction(ctx, "viewer", "example.thing.read")
	must(err, "AddAction viewer")
	agree("roles with no grants")

	_, err = pm.CreateGrant(ctx, models.Grant{Subject: "member-a", Scope: "instance:alpha", RoleCode: "editor", GrantedBy: "admin"})
	must(err, "grant editor alpha")
	_, err = pm.CreateGrant(ctx, models.Grant{Subject: "member-a", Scope: "*", RoleCode: "viewer", GrantedBy: "admin"})
	must(err, "grant viewer everywhere")
	_, err = pm.CreateGrant(ctx, models.Grant{Subject: "member-b", Scope: "instance:beta", RoleCode: "viewer", GrantedBy: "admin"})
	must(err, "grant viewer beta")
	agree("scoped and everywhere grants")

	ok, err := pm.MayPerform(ctx, "member-a", "instance:alpha", "example.thing.delete")
	if err != nil || !ok {
		t.Fatalf("precondition: member-a's editor grant authorizes delete in alpha: ok=%v err=%v", ok, err)
	}

	must(pm.RemoveAction(ctx, "editor", "example.thing.delete"), "RemoveAction")
	agree("an action removed from a granted role")

	_, err = pm.SetActions(ctx, "editor", []string{"example.thing.update", "example.other.read"})
	must(err, "SetActions replace")
	agree("a granted role's action list replaced")

	must(pm.WithdrawGrant(ctx, "member-a", "*", "viewer"), "WithdrawGrant everywhere")
	agree("the everywhere grant withdrawn")

	_, err = pm.CreateGrant(ctx, models.Grant{Subject: "member-a", Scope: "*", RoleCode: "viewer", GrantedBy: "admin"})
	must(err, "reinstate viewer everywhere")
	agree("the everywhere grant reinstated")

	must(pm.DeleteRole(ctx, "editor"), "DeleteRole editor")
	agree("a granted role retired")

	ok, err = pm.MayPerform(ctx, "member-a", "instance:alpha", "example.thing.update")
	if err != nil || ok {
		t.Fatalf("precondition: a retired role authorizes nothing: ok=%v err=%v", ok, err)
	}
}
