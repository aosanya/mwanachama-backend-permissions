package mwanachamapermissions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
)

const (
	recordAction = "forms.answer.submit"
	matterOne    = "record:forms/form/f-1"
	matterTwo    = "record:forms/form/f-2"
)

func seedOutsider(t *testing.T, pm permissions.PermissionsManager) {
	t.Helper()
	ctx := context.Background()
	if _, err := pm.CreateRole(ctx, models.Role{Code: "outsider", Name: "Outsider"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := pm.AddAction(ctx, "outsider", recordAction); err != nil {
		t.Fatalf("AddAction: %v", err)
	}
}

func mayOn(t *testing.T, pm permissions.PermissionsManager, scope string) bool {
	t.Helper()
	ok, err := pm.MayPerform(context.Background(), "cleo", scope, recordAction)
	if err != nil {
		t.Fatalf("MayPerform(%s): %v", scope, err)
	}
	return ok
}

func TestARecordGrantAnswersForThatRecordAlone(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	seedOutsider(t, pm)
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "cleo", Scope: matterOne, RoleCode: "outsider"}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	if !mayOn(t, pm, matterOne) {
		t.Errorf("a grant on %s refused a check on the same record", matterOne)
	}
	if mayOn(t, pm, matterTwo) {
		t.Errorf("a grant on %s answered for a different record", matterOne)
	}
	if mayOn(t, pm, "module:forms") {
		t.Errorf("a record grant satisfied a check on the whole module")
	}
	if mayOn(t, pm, "record:forms/form") {
		t.Errorf("a record grant satisfied a check on a prefix of its record")
	}
	if got, _ := pm.ActionsFor(ctx, "cleo", "module:forms"); len(got) != 0 {
		t.Errorf("ActionsFor on the module listed %v from a record grant", got)
	}
}

func TestAMalformedRecordScopeIsRefused(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	seedOutsider(t, pm)
	for _, scope := range []string{
		"record:forms",
		"record:forms/form",
		"record:forms/form/",
		"record:forms/form/f-1/extra",
		"record:Forms/form/f-1",
		"record:forms//f-1",
		"record:forms/form/*",
	} {
		_, err := pm.CreateGrant(ctx, models.Grant{Subject: "cleo", Scope: scope, RoleCode: "outsider"})
		if !errors.Is(err, permissions.ErrInvalidScope) {
			t.Errorf("CreateGrant(%q) = %v, want ErrInvalidScope", scope, err)
		}
	}
}

func TestAGrantStopsAnsweringAtItsEnd(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	seedOutsider(t, pm)
	end := time.Now().Add(time.Second).Format(time.RFC3339Nano)
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "cleo", Scope: matterOne, RoleCode: "outsider", EndsAt: end}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if !mayOn(t, pm, matterOne) {
		t.Fatalf("a grant before its end refused its record")
	}
	if got, _ := pm.ListGrants(ctx, models.GrantFilter{Subject: "cleo"}); len(got) != 1 {
		t.Fatalf("ListGrants before the end = %d grants, want 1", len(got))
	}

	time.Sleep(1200 * time.Millisecond)

	if mayOn(t, pm, matterOne) {
		t.Errorf("a grant past its end still answered")
	}
	if got, _ := pm.ActionsAnywhere(ctx, "cleo"); len(got) != 0 {
		t.Errorf("ActionsAnywhere past the end = %v, want none", got)
	}
	if got, _ := pm.ListGrants(ctx, models.GrantFilter{Subject: "cleo"}); len(got) != 0 {
		t.Errorf("ListGrants past the end = %v, want none", got)
	}
}

func TestAnEndAlreadyPassedOrUnreadableIsRefused(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	seedOutsider(t, pm)
	for _, end := range []string{
		time.Now().Add(-time.Minute).Format(time.RFC3339),
		"tomorrow",
		"2099-13-01T00:00:00Z",
	} {
		_, err := pm.CreateGrant(ctx, models.Grant{Subject: "cleo", Scope: matterOne, RoleCode: "outsider", EndsAt: end})
		if !errors.Is(err, permissions.ErrInvalidEnd) {
			t.Errorf("CreateGrant(ends_at %q) = %v, want ErrInvalidEnd", end, err)
		}
	}
}

func TestAnEndIsStoredInUTCAndWithdrawalStillWorks(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	seedOutsider(t, pm)
	g, err := pm.CreateGrant(ctx, models.Grant{Subject: "cleo", Scope: matterOne, RoleCode: "outsider", EndsAt: "2099-06-01T12:00:00+03:00"})
	if err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if g.EndsAt != "2099-06-01T09:00:00.000000000Z" {
		t.Errorf("EndsAt = %q, want the UTC fixed-width form", g.EndsAt)
	}
	if !mayOn(t, pm, matterOne) {
		t.Fatalf("a grant ending in 2099 refused its record")
	}
	if err := pm.WithdrawGrant(ctx, "cleo", matterOne, "outsider"); err != nil {
		t.Fatalf("WithdrawGrant: %v", err)
	}
	if mayOn(t, pm, matterOne) {
		t.Errorf("a withdrawn record grant still answered")
	}
}
