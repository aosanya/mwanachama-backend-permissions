package mwanachamapermissions_test

import (
	"context"
	"errors"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestAccessOfRefusesAPartitionThatIsEmptyOrMalformed(t *testing.T) {
	pm := newTestManager(t)
	for _, partition := range []string{"", "agency a", "-agency", "a/b"} {
		lines, err := pm.AccessOf(context.Background(), "asha", partition)
		if !errors.Is(err, permissions.ErrInvalidPartition) {
			t.Errorf("AccessOf(asha, %q) = %v, %v; want ErrInvalidPartition", partition, lines, err)
		}
	}
}

func TestAccessOfWithNoSubjectIsEmptyAndNeverNil(t *testing.T) {
	pm := newTestManager(t)
	lines, err := pm.AccessOf(context.Background(), "", "agency-a")
	if err != nil || lines == nil || len(lines) != 0 {
		t.Errorf("AccessOf(\"\", agency-a) = %#v, %v; want an empty non-nil list", lines, err)
	}
}

func TestAccessOfListsExactlyWhatTheGateAllowsInThatPartition(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	for _, r := range []struct {
		code    string
		actions []string
	}{
		{"farmer", []string{"forms.answer.submit", "shop.item.read"}},
		{"member", []string{"chat.thread.read"}},
		{"outsider", []string{"outside.thing.read"}},
	} {
		if _, err := pm.CreateRole(ctx, models.Role{Code: r.code, Name: r.code}); err != nil {
			t.Fatalf("create role %s: %v", r.code, err)
		}
		if _, err := pm.SetActions(ctx, r.code, r.actions); err != nil {
			t.Fatalf("set actions %s: %v", r.code, err)
		}
	}
	for _, g := range []models.Grant{
		{Subject: "asha", Scope: "module:forms", Partition: "agency-a", RoleCode: "farmer"},
		{Subject: "asha", Scope: "*", RoleCode: "member"},
		{Subject: "asha", Scope: "module:outside", Partition: "agency-b", RoleCode: "outsider"},
	} {
		if _, err := pm.CreateGrant(ctx, g); err != nil {
			t.Fatalf("grant %s: %v", g.RoleCode, err)
		}
	}

	allActions := []string{"forms.answer.submit", "shop.item.read", "chat.thread.read", "outside.thing.read", "nothing.thing.read"}
	for _, partition := range []string{"agency-a", "agency-b", "agency-c"} {
		lines, err := pm.AccessOf(ctx, "asha", partition)
		if err != nil {
			t.Fatalf("AccessOf(asha, %s): %v", partition, err)
		}
		listed := map[string]bool{}
		for _, l := range lines {
			listed[l.Action] = true
			ok, err := pm.MayPerformIn(ctx, "asha", partition, l.Scope, l.Action)
			if err != nil || !ok {
				t.Errorf("%s lists %s in %s, which the gate refuses (%v)", partition, l.Action, l.Scope, err)
			}
		}
		for _, action := range allActions {
			for _, scope := range []string{"module:forms", "module:outside", "*"} {
				ok, err := pm.MayPerformIn(ctx, "asha", partition, scope, action)
				if err != nil {
					t.Fatalf("MayPerformIn: %v", err)
				}
				if ok && !listed[action] {
					t.Errorf("%s: the gate allows %s in %s but AccessOf does not list it", partition, action, scope)
				}
			}
		}
	}
}
