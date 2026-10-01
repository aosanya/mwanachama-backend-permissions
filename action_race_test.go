package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestPM11_OpenBug_ConcurrentAddActionOnTheSamePairFailsInstead(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:pm11?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(2)
	t.Cleanup(func() { sqlDB.Close() })

	s, err := permissions.LoadSpec("spec/examples/wakala.permissions.json")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if err := permissions.Migrate(db, s); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mgr, err := permissions.NewPermissionsManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, models.Role{Code: "writer", Name: "Writer"}); err != nil {
		t.Fatalf("create role: %v", err)
	}

	raRole, ok := s.ByRole("role_action")
	if !ok {
		t.Fatal("the shipped spec fills no role_action role")
	}
	raTable := s.TableFor(raRole)

	fired := false
	var raceErr error
	err = db.Callback().Create().Before("gorm:create").Register("pm11_concurrent_add_action", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != raTable {
			return
		}
		fired = true
		_, raceErr = mgr.AddAction(context.Background(), "writer", "catalog.entry.upsert")
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, outerErr := mgr.AddAction(ctx, "writer", "catalog.entry.upsert")
	if !fired {
		t.Fatalf("precondition: AddAction never reached its insert")
	}
	if raceErr != nil {
		t.Fatalf("precondition: the concurrent AddAction that won the race failed: %v", raceErr)
	}
	if outerErr != nil {
		t.Errorf("PM11 open bug: AddAction(\"writer\", \"catalog.entry.upsert\") failed with %v while a concurrent AddAction granting the same role the same action was committing; want it to succeed, because two legitimate callers granting the same permission for the first time at once should both see their write land, not surface a raw SQL constraint failure to one of them", outerErr)
	}

	actions, err := mgr.ListActions(ctx, "writer")
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}
	if len(actions) != 1 {
		t.Errorf("PM11 open bug: after the race, ListActions(\"writer\") returned %d rows, want exactly 1 — both racing callers asked to grant the identical (role, action) pair", len(actions))
	}
}
