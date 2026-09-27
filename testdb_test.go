package mwanachamapermissions_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func newTestManager(t *testing.T) permissions.PermissionsManager {
	t.Helper()
	mgr, _ := newTestManagerSpec(t)
	return mgr
}

func newTestManagerSpec(t *testing.T) (permissions.PermissionsManager, *spec.Spec) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := permissions.LoadSpec("spec/examples/wakala.permissions.json")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if err := permissions.Migrate(db, s); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mgr, err := permissions.NewPermissionsManager(db, s)
	if err != nil {
		t.Fatalf("NewPermissionsManager: %v", err)
	}
	return mgr, s
}
