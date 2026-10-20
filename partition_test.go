package mwanachamapermissions_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
)

const partitionedAction = "forms.answer.submit"

func seedAnswerer(t *testing.T, pm permissions.PermissionsManager) {
	t.Helper()
	ctx := context.Background()
	if _, err := pm.CreateRole(ctx, models.Role{Code: "answerer", Name: "Answerer"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := pm.AddAction(ctx, "answerer", partitionedAction); err != nil {
		t.Fatalf("AddAction: %v", err)
	}
}

func mayIn(t *testing.T, pm permissions.PermissionsManager, subject, partition string) bool {
	t.Helper()
	ok, err := pm.MayPerformIn(context.Background(), subject, partition, "module:forms", partitionedAction)
	if err != nil {
		t.Fatalf("MayPerformIn(%s, %q): %v", subject, partition, err)
	}
	return ok
}

func TestAGrantInOnePartitionNeverAuthorizesARequestInAnother(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	seedAnswerer(t, pm)
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "ann", Scope: "module:forms", Partition: "alpha", RoleCode: "answerer"}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	if !mayIn(t, pm, "ann", "alpha") {
		t.Errorf("a grant naming partition alpha refused a request in alpha")
	}
	if mayIn(t, pm, "ann", "beta") {
		t.Errorf("a grant naming partition alpha authorized a request in beta")
	}
	if mayIn(t, pm, "ann", "") {
		t.Errorf("a grant naming partition alpha authorized a request outside every partition")
	}
	if ok, err := pm.MayPerform(ctx, "ann", "module:forms", partitionedAction); err != nil || ok {
		t.Errorf("MayPerform, which asks outside every partition, answered %v %v for a grant naming alpha", ok, err)
	}
	if listed, err := pm.ActionsFor(ctx, "ann", "module:forms"); err != nil || len(listed) != 0 {
		t.Errorf("ActionsFor, which lists outside every partition, answered %v %v for a grant naming alpha", listed, err)
	}
}

func TestAGrantNamingNoPartitionStillHoldsInEveryPartition(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	seedAnswerer(t, pm)
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "ann", Scope: "module:forms", RoleCode: "answerer"}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	for _, partition := range []string{"", "alpha", "beta"} {
		if !mayIn(t, pm, "ann", partition) {
			t.Errorf("a grant naming no partition refused a request in %q", partition)
		}
	}
	if ok, err := pm.MayPerform(ctx, "ann", "module:forms", partitionedAction); err != nil || !ok {
		t.Errorf("MayPerform answered %v %v for a grant naming no partition", ok, err)
	}
}

func TestOneSubjectHoldsTheSameRoleInTwoPartitionsAndLosesOnlyTheOneWithdrawn(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	seedAnswerer(t, pm)
	for _, partition := range []string{"alpha", "beta"} {
		if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "ann", Scope: "module:forms", Partition: partition, RoleCode: "answerer"}); err != nil {
			t.Fatalf("CreateGrant(%s): %v", partition, err)
		}
	}
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "ann", Scope: "module:forms", Partition: "alpha", RoleCode: "answerer"}); !errors.Is(err, permissions.ErrDuplicateGrant) {
		t.Errorf("a second grant in alpha answered %v, want ErrDuplicateGrant", err)
	}
	if got, err := pm.ListGrants(ctx, models.GrantFilter{Subject: "ann", Partition: "beta"}); err != nil || len(got) != 1 || got[0].Partition != "beta" {
		t.Errorf("ListGrants(partition=beta) answered %+v %v", got, err)
	}

	if err := pm.WithdrawGrant(ctx, "ann", "module:forms", "answerer"); !errors.Is(err, permissions.ErrGrantNotFound) {
		t.Errorf("withdrawing the grant naming no partition answered %v, want ErrGrantNotFound: only alpha's and beta's exist", err)
	}
	if err := pm.WithdrawGrantIn(ctx, "ann", "alpha", "module:forms", "answerer"); err != nil {
		t.Fatalf("WithdrawGrantIn(alpha): %v", err)
	}
	if mayIn(t, pm, "ann", "alpha") {
		t.Errorf("the grant withdrawn in alpha still authorizes there")
	}
	if !mayIn(t, pm, "ann", "beta") {
		t.Errorf("withdrawing in alpha took beta's grant with it")
	}
}

func TestAMalformedPartitionIsRefused(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	seedAnswerer(t, pm)
	for _, partition := range []string{" alpha", "al pha", "alpha'--", "-alpha", "alpha/beta"} {
		if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "ann", Scope: "module:forms", Partition: partition, RoleCode: "answerer"}); !errors.Is(err, permissions.ErrInvalidPartition) {
			t.Errorf("CreateGrant(partition=%q) answered %v, want ErrInvalidPartition", partition, err)
		}
		if _, err := pm.MayPerformIn(ctx, "ann", partition, "module:forms", partitionedAction); !errors.Is(err, permissions.ErrInvalidPartition) {
			t.Errorf("MayPerformIn(partition=%q) answered %v, want ErrInvalidPartition", partition, err)
		}
	}
}

func TestMigrateGivesAGrantTableFromBeforePartitionsOneAndItsGrantsKeepHoldingEverywhere(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := permissions.LoadSpec("spec/examples/wakala.permissions.json")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	o, _ := s.ByRole("grant")
	grants := s.TableFor(o)
	legacy := fmt.Sprintf(`create table %s (
  subject text not null,
  scope text not null,
  role_code text not null,
  granted_by text default '',
  granted_at text default '',
  deleted boolean default false,
  primary key (subject, scope, role_code)
)`, grants)
	if err := db.Exec(legacy).Error; err != nil {
		t.Fatalf("create legacy grants: %v", err)
	}
	if err := db.Exec(fmt.Sprintf("insert into %s (subject, scope, role_code) values ('ann', 'module:forms', 'answerer')", grants)).Error; err != nil {
		t.Fatalf("insert legacy grant: %v", err)
	}

	if err := permissions.Migrate(db, s); err != nil {
		t.Fatalf("Migrate over the legacy table: %v", err)
	}
	if !db.Migrator().HasColumn(grants, "partition") {
		t.Fatalf("Migrate left %s without a partition column", grants)
	}
	pm, err := permissions.NewPermissionsManager(db, s)
	if err != nil {
		t.Fatalf("NewPermissionsManager: %v", err)
	}
	if _, err := pm.CreateRole(ctx, models.Role{Code: "answerer", Name: "Answerer"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := pm.AddAction(ctx, "answerer", partitionedAction); err != nil {
		t.Fatalf("AddAction: %v", err)
	}
	for _, partition := range []string{"", "alpha"} {
		if !mayIn(t, pm, "ann", partition) {
			t.Errorf("a grant made before partitions existed refused a request in %q", partition)
		}
	}
	if err := permissions.Migrate(db, s); err != nil {
		t.Errorf("a second Migrate: %v", err)
	}
}
