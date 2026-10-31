package mwanachamapermissions_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestMigrateCarriesAPowersTableAndItsPowerColumnOverToPermissions(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := permissions.LoadSpec("spec/examples/wakala.permissions.json")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	o, _ := s.ByRole("permission")
	former := o
	former.Table = "powers"
	formerTable := s.TableFor(former)
	create := fmt.Sprintf(`create table %s (
  partition text not null,
  power text not null,
  role_kind text not null,
  layer text not null,
  held boolean default false,
  set_by text default '',
  set_at text default '',
  primary key (partition, power, role_kind, layer)
)`, formerTable)
	if err := db.Exec(create).Error; err != nil {
		t.Fatalf("create powers table: %v", err)
	}
	insert := fmt.Sprintf("insert into %s (partition, power, role_kind, layer, held) values ('agy1', 'run_poll', 'treasurer', 'definition', true)", formerTable)
	if err := db.Exec(insert).Error; err != nil {
		t.Fatalf("insert power: %v", err)
	}

	if err := permissions.Migrate(db, s); err != nil {
		t.Fatalf("Migrate over the powers table: %v", err)
	}
	if db.Migrator().HasTable(formerTable) {
		t.Errorf("Migrate left the powers table %s behind", formerTable)
	}
	if !db.Migrator().HasColumn(s.TableFor(o), "permission") || db.Migrator().HasColumn(s.TableFor(o), "power") {
		t.Errorf("Migrate did not rename the power column to permission")
	}
	mgr, err := permissions.NewPermissionsManager(db, s)
	if err != nil {
		t.Fatalf("NewPermissionsManager: %v", err)
	}
	got, err := mgr.ReadPermission(ctx, "agy1", "run_poll", "treasurer")
	if err != nil || !got.Held || got.Layer != models.PermissionLayerDefinition {
		t.Fatalf("ReadPermission after the rename = %+v, %v; want the carried-over tick", got, err)
	}
	if err := permissions.Migrate(db, s); err != nil {
		t.Errorf("a second Migrate: %v", err)
	}
}
