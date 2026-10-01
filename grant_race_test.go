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

func TestPM12_OpenBug_ConcurrentCreateGrantOnTheSameTripleFailsInstead(t *testing.T) {
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

	grantRole, ok := s.ByRole("grant")
	if !ok {
		t.Fatal("the shipped spec fills no grant role")
	}
	grantTable := s.TableFor(grantRole)

	fired := false
	var raceErr error
	err = db.Callback().Create().Before("gorm:create").Register("pm12_concurrent_create_grant", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != grantTable {
			return
		}
		fired = true
		_, raceErr = mgr.CreateGrant(context.Background(), models.Grant{Subject: "bob", Scope: "instance:abc", RoleCode: "writer"})
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, outerErr := mgr.CreateGrant(ctx, models.Grant{Subject: "bob", Scope: "instance:abc", RoleCode: "writer"})
	if !fired {
		t.Fatalf("precondition: CreateGrant never reached its insert")
	}
	if raceErr != nil {
		t.Fatalf("precondition: the concurrent CreateGrant that won the race failed: %v", raceErr)
	}
	if outerErr != nil {
		t.Errorf("PM12 open bug: CreateGrant(bob, instance:abc, writer) failed with %v while a concurrent CreateGrant for the identical (subject, scope, role_code) triple was committing; want it to answer ErrDuplicateGrant, not surface a raw SQL constraint failure to one of two legitimate callers granting the same permission for the first time at once", outerErr)
	}

	grants, err := mgr.ListGrants(ctx, models.GrantFilter{Subject: "bob"})
	if err != nil {
		t.Fatalf("list grants: %v", err)
	}
	if len(grants) != 1 {
		t.Errorf("PM12 open bug: after the race, ListGrants(subject=bob) returned %d rows, want exactly 1 — both racing callers asked to create the identical grant", len(grants))
	}
}
