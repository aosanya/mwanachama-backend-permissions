package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM9_OpenBug_SetActionsRejectsADuplicateActionInTheList(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, models.Role{Code: "writer", Name: "Writer"}); err != nil {
		t.Fatalf("create role: %v", err)
	}

	got, err := mgr.SetActions(ctx, "writer", []string{"catalog.entry.upsert", "catalog.entry.upsert"})
	if err != nil {
		t.Fatalf("PM9 open bug: SetActions with a duplicate action in its own input list failed instead of deduplicating it: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("PM9 open bug: SetActions(writer, [catalog.entry.upsert, catalog.entry.upsert]) returned %d actions, want 1 deduplicated action", len(got))
	}

	acts, err := mgr.ListActions(ctx, "writer")
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}
	if len(acts) != 1 {
		t.Fatalf("PM9 open bug: after SetActions with a duplicated action, ListActions returned %d rows, want exactly 1 — the caller's redundant entry should never reach the store twice", len(acts))
	}
}
