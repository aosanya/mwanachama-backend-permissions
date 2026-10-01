package mwanachamapermissions_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestPM12_OpenBug_ConcurrentCreateRoleOnANewCodeFailsInstead(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:pm12?mode=memory&cache=shared"), &gorm.Config{
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

	roleRole, ok := s.ByRole("role")
	if !ok {
		t.Fatal("the shipped spec fills no role role")
	}
	roleTable := s.TableFor(roleRole)

	fired := false
	var raceErr error
	err = db.Callback().Create().Before("gorm:create").Register("pm12_concurrent_create_role", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != roleTable {
			return
		}
		fired = true
		_, raceErr = mgr.CreateRole(context.Background(), permissions.Role{Code: "writer", Name: "Writer-B"})
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, outerErr := mgr.CreateRole(ctx, permissions.Role{Code: "writer", Name: "Writer-A"})
	if !fired {
		t.Fatalf("precondition: CreateRole never reached its insert")
	}
	if raceErr != nil {
		t.Fatalf("precondition: the concurrent CreateRole that won the race failed: %v", raceErr)
	}
	if outerErr != nil {
		t.Errorf("PM12 open bug: CreateRole(code=writer) failed with %v while a concurrent CreateRole for the same brand-new code was committing; want it to succeed or answer the domain ErrDuplicateRole, not surface a raw SQL constraint failure to one of two legitimate callers declaring the same bundle at once", outerErr)
	}

	roles, err := mgr.ListRoles(ctx, permissions.RoleFilter{Code: "writer"})
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(roles) != 1 {
		t.Errorf("PM12 open bug: after the race, code \"writer\" names %d live roles, want exactly 1", len(roles))
	}
}
