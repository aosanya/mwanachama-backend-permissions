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

func TestTheStoreRefusesASecondGrantRowForOneSubjectScopeAndRole_SoOneWithdrawalRevokesIt(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	pm, err := permissions.NewPermissionsManager(db, s)
	if err != nil {
		t.Fatalf("NewPermissionsManager: %v", err)
	}

	if _, err := pm.CreateRole(ctx, models.Role{Code: "writer", Name: "Writer"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := pm.AddAction(ctx, "writer", "catalog.entry.upsert"); err != nil {
		t.Fatalf("AddAction: %v", err)
	}
	if _, err := pm.CreateGrant(ctx, models.Grant{Subject: "bob", Scope: "module:catalog", RoleCode: "writer", GrantedBy: "admin"}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if ok, err := pm.MayPerform(ctx, "bob", "module:catalog", "catalog.entry.upsert"); err != nil || !ok {
		t.Fatalf("precondition: bob's grant should permit the action, got %v, %v", ok, err)
	}

	second := map[string]any{
		"subject": "bob", "scope": "module:catalog", "role_code": "writer",
		"granted_by": "racer", "granted_at": "2026-09-30T00:00:00Z", "deleted": false,
	}
	if err := db.WithContext(ctx).Table("agency_permissions_grants").Create(second).Error; err == nil {
		t.Errorf("a second live grant row for (bob, module:catalog, writer) was stored past CreateGrant's pre-check; a racing grant would survive the withdrawal of the first")
	}
	var rows int64
	if err := db.WithContext(ctx).Table("agency_permissions_grants").Where("subject = ? and scope = ? and role_code = ?", "bob", "module:catalog", "writer").Count(&rows).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Errorf("the store holds %d grant rows for one subject, scope and role, want 1", rows)
	}

	if err := pm.WithdrawGrant(ctx, "bob", "module:catalog", "writer"); err != nil {
		t.Fatalf("WithdrawGrant: %v", err)
	}
	if ok, err := pm.MayPerform(ctx, "bob", "module:catalog", "catalog.entry.upsert"); err != nil || ok {
		t.Errorf("after one withdrawal bob may still perform the action: %v, %v, want false", ok, err)
	}
}
