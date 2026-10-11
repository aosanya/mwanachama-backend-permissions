package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestAGrantToTheSubjectStarAuthorizesOnlyTheLiteralSubjectStar(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)

	if _, err := pm.CreateRole(ctx, models.Role{Code: "editor", Name: "Editor"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := pm.AddAction(ctx, "editor", "example.thing.update"); err != nil {
		t.Fatalf("AddAction: %v", err)
	}
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: models.ScopeEverywhere, Scope: models.ScopeEverywhere, RoleCode: "editor", GrantedBy: "admin"}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	ok, err := pm.MayPerform(ctx, models.ScopeEverywhere, "instance:alpha", "example.thing.update")
	if err != nil || !ok {
		t.Fatalf("precondition: the grant authorizes its own literal subject: ok=%v err=%v", ok, err)
	}

	for _, subject := range []string{"member-a", "anonymous", "%", "**", " *"} {
		if ok, err := pm.MayPerform(ctx, subject, "instance:alpha", "example.thing.update"); err != nil || ok {
			t.Errorf("a grant to subject * authorized subject %q: ok=%v err=%v", subject, ok, err)
		}
		if got, err := pm.ActionsFor(ctx, subject, "instance:alpha"); err != nil || len(got) != 0 {
			t.Errorf("a grant to subject * listed actions for subject %q: %v err=%v", subject, got, err)
		}
		if got, err := pm.ActionsAnywhere(ctx, subject); err != nil || len(got) != 0 {
			t.Errorf("a grant to subject * listed actions anywhere for subject %q: %v err=%v", subject, got, err)
		}
	}
}
