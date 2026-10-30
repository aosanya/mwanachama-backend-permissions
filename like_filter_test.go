package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM7_NameFilterStillFindsNamesHoldingTheWildcardsAndTheEscape(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	names := map[string]string{"r0": "100% Owner", "r1": "Alpha_Beta", "r2": "Wow! Lead", "r3": "Plain"}
	for code, name := range names {
		if _, err := mgr.CreateRole(ctx, models.Role{Code: code, Name: name}); err != nil {
			t.Fatalf("CreateRole(%q): %v", name, err)
		}
	}

	for search, want := range map[string]string{"0% o": "r0", "ha_be": "r1", "w! l": "r2", "plain": "r3"} {
		got, err := mgr.ListRoles(ctx, models.RoleFilter{Name: search})
		if err != nil {
			t.Fatalf("ListRoles(%q): %v", search, err)
		}
		if len(got) != 1 || got[0].Code != want {
			t.Errorf("ListRoles(name=%q) = %v; want only role %s", search, got, want)
		}
	}
}
