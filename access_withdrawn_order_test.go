package mwanachamapermissions_test

import (
	"context"
	"reflect"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func readerManager(t *testing.T) permissions.PermissionsManager {
	t.Helper()
	pm := newTestManager(t)
	ctx := context.Background()
	for _, r := range []struct {
		code, action string
	}{
		{"reader", "library.book.read"},
		{"writer", "library.book.write"},
	} {
		if _, err := pm.CreateRole(ctx, models.Role{Code: r.code, Name: r.code}); err != nil {
			t.Fatalf("precondition: create role %s: %v", r.code, err)
		}
		if _, err := pm.SetActions(ctx, r.code, []string{r.action}); err != nil {
			t.Fatalf("precondition: set actions %s: %v", r.code, err)
		}
	}
	return pm
}

func grantAll(t *testing.T, pm permissions.PermissionsManager, grants ...models.Grant) {
	t.Helper()
	for _, g := range grants {
		if _, err := pm.CreateGrant(context.Background(), g); err != nil {
			t.Fatalf("precondition: grant %s in %q on %s: %v", g.RoleCode, g.Partition, g.Scope, err)
		}
	}
}

func scopesAndPartitions(lines []permissions.AccessLine) [][2]string {
	out := make([][2]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, [2]string{l.Scope, l.Partition})
	}
	return out
}

func TestAccessOfLeavesOutAWithdrawnGrantAndKeepsTheOnesBesideIt(t *testing.T) {
	ctx := context.Background()
	pm := readerManager(t)
	grantAll(t, pm,
		models.Grant{Subject: "asha", Scope: "module:library", Partition: "agency-a", RoleCode: "reader"},
		models.Grant{Subject: "asha", Scope: "module:library", Partition: "agency-a", RoleCode: "writer"},
	)

	before, err := pm.AccessOf(ctx, "asha", "agency-a")
	if err != nil || len(before) != 2 {
		t.Fatalf("precondition: AccessOf before the withdrawal = %+v, %v; want both lines", before, err)
	}
	if err := pm.WithdrawGrantIn(ctx, "asha", "agency-a", "module:library", "reader"); err != nil {
		t.Fatalf("precondition: withdraw: %v", err)
	}

	got, err := pm.AccessOf(ctx, "asha", "agency-a")
	if err != nil {
		t.Fatalf("AccessOf: %v", err)
	}
	if len(got) != 1 || got[0].RoleCode != "writer" || got[0].Action != "library.book.write" {
		t.Errorf("AccessOf after withdrawing reader = %+v, want only the writer line: a withdrawn grant permits nothing", got)
	}
	allowed, err := pm.MayPerformIn(ctx, "asha", "agency-a", "module:library", "library.book.read")
	if err != nil || allowed {
		t.Errorf("MayPerformIn after the withdrawal = %v, %v; want false, so AccessOf lists exactly what the gate allows", allowed, err)
	}
}

func TestAccessOfListsAReinstatedGrantAgainAfterItWasWithdrawn(t *testing.T) {
	ctx := context.Background()
	pm := readerManager(t)
	grant := models.Grant{Subject: "asha", Scope: "module:library", Partition: "agency-a", RoleCode: "reader"}
	grantAll(t, pm, grant)
	if err := pm.WithdrawGrantIn(ctx, "asha", "agency-a", "module:library", "reader"); err != nil {
		t.Fatalf("precondition: withdraw: %v", err)
	}
	grantAll(t, pm, grant)

	got, err := pm.AccessOf(ctx, "asha", "agency-a")
	if err != nil || len(got) != 1 {
		t.Errorf("AccessOf after reinstating = %+v, %v; want the one line back", got, err)
	}
}

func TestAccessOfBreaksATieOnActionAndRoleByScopeBeforeByPartition(t *testing.T) {
	pm := readerManager(t)
	grantAll(t, pm,
		models.Grant{Subject: "asha", Scope: "module:zebra", RoleCode: "reader"},
		models.Grant{Subject: "asha", Scope: "module:mango", Partition: "agency-a", RoleCode: "reader"},
		models.Grant{Subject: "asha", Scope: "module:apple", Partition: "agency-a", RoleCode: "reader"},
	)

	lines, err := pm.AccessOf(context.Background(), "asha", "agency-a")
	if err != nil {
		t.Fatalf("AccessOf: %v", err)
	}
	want := [][2]string{
		{"module:apple", "agency-a"},
		{"module:mango", "agency-a"},
		{"module:zebra", ""},
	}
	if got := scopesAndPartitions(lines); !reflect.DeepEqual(got, want) {
		t.Errorf("lines tied on action and role came as %v, want them by scope, whatever their partition, %v", got, want)
	}
}

func TestAccessOfBreaksATieOnActionRoleAndScopeByPartitionUnpartitionedFirst(t *testing.T) {
	pm := readerManager(t)
	grantAll(t, pm,
		models.Grant{Subject: "asha", Scope: "module:library", Partition: "agency-a", RoleCode: "reader"},
		models.Grant{Subject: "asha", Scope: "module:library", RoleCode: "reader"},
	)

	lines, err := pm.AccessOf(context.Background(), "asha", "agency-a")
	if err != nil {
		t.Fatalf("AccessOf: %v", err)
	}
	want := [][2]string{
		{"module:library", ""},
		{"module:library", "agency-a"},
	}
	if got := scopesAndPartitions(lines); !reflect.DeepEqual(got, want) {
		t.Errorf("lines tied on action, role and scope came as %v, want the unpartitioned one first %v", got, want)
	}
}
