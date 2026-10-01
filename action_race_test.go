package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestPM13_OpenBug_ConcurrentAddActionOnTheSamePairFailsInstead(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:pm13?mode=memory&cache=shared"), &gorm.Config{
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
		t.Fatalf("load spec: %v", err)
	}
	if err := permissions.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mgr, err := permissions.NewPermissionsManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	ctx := context.Background()

	if _, err := mgr.CreateRole(ctx, permissions.Role{Code: "writer", Name: "Writer"}); err != nil {
		t.Fatalf("create role: %v", err)
	}

	actionRole, ok := s.ByRole("role_action")
	if !ok {
		t.Fatal("the shipped spec fills no role_action role")
	}
	actionTable := s.TableFor(actionRole)

	fired := false
	var raceErr error
	err = db.Callback().Create().Before("gorm:create").Register("pm13_concurrent_add_action", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != actionTable {
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
		t.Errorf("PM13 open bug: AddAction(writer, catalog.entry.upsert) failed with %v while a concurrent AddAction for the same (role_code, action) pair was committing; want it to succeed as a no-op, the same way AddAction already treats a repeat call that is not racing another one", outerErr)
	}

	actions, err := mgr.ListActions(ctx, "writer")
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}
	if len(actions) != 1 {
		t.Errorf("PM13 open bug: after the race, role \"writer\" holds %d actions, want exactly 1", len(actions))
	}
}
