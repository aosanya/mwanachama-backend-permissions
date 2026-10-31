package mwanachamapermissions_test

import (
	"context"
	"errors"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func TestAnOverrideWinsOverTheDefinitionAndSaysWhoSetIt(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	key := models.Permission{Partition: "agy1", Permission: "run_poll", RoleKind: "treasurer"}

	def := key
	def.Held = true
	if _, err := mgr.SetPermissionDefinition(ctx, def); err != nil {
		t.Fatalf("definition: %v", err)
	}
	got, err := mgr.ReadPermission(ctx, "agy1", "run_poll", "treasurer")
	if err != nil || !got.Held || got.Layer != models.PermissionLayerDefinition || got.SetBy != "" {
		t.Fatalf("read definition: %+v %v", got, err)
	}

	over := key
	over.Held, over.SetBy = false, "owner-1"
	if _, err := mgr.OverridePermission(ctx, over); err != nil {
		t.Fatalf("override: %v", err)
	}
	got, err = mgr.ReadPermission(ctx, "agy1", "run_poll", "treasurer")
	if err != nil || got.Held || got.Layer != models.PermissionLayerOverride || got.SetBy != "owner-1" || got.SetAt == "" {
		t.Fatalf("read override: %+v %v", got, err)
	}

	def.Held = false
	if _, err := mgr.SetPermissionDefinition(ctx, def); err != nil {
		t.Fatalf("republish: %v", err)
	}
	if got, _ := mgr.ReadPermission(ctx, "agy1", "run_poll", "treasurer"); got.Layer != models.PermissionLayerOverride {
		t.Fatalf("a republish displaced the override: %+v", got)
	}

	if err := mgr.ClearPermissionOverride(ctx, "agy1", "run_poll", "treasurer"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, err = mgr.ReadPermission(ctx, "agy1", "run_poll", "treasurer")
	if err != nil || got.Layer != models.PermissionLayerDefinition || got.Held {
		t.Fatalf("after clear: %+v %v", got, err)
	}
}

func TestAPermissionIsHeldPerPartitionAndRoleKind(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	if _, err := mgr.SetPermissionDefinition(ctx, models.Permission{Partition: "agy1", Permission: "post_notice", RoleKind: "secretary", Held: true}); err != nil {
		t.Fatal(err)
	}
	for _, k := range [][3]string{{"agy2", "post_notice", "secretary"}, {"agy1", "post_notice", "chair"}, {"agy1", "post_event", "secretary"}} {
		if _, err := mgr.ReadPermission(ctx, k[0], k[1], k[2]); !errors.Is(err, permissions.ErrPermissionNotFound) {
			t.Fatalf("%v: want ErrPermissionNotFound, got %v", k, err)
		}
	}
}

func TestAnOverrideNeedsWhoSetIt(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.OverridePermission(context.Background(), models.Permission{Partition: "agy1", Permission: "run_poll", RoleKind: "chair", Held: true})
	if !errors.Is(err, permissions.ErrInvalidPermission) {
		t.Fatalf("want ErrInvalidPermission, got %v", err)
	}
}

func TestAPermissionSettingRefusesAMalformedKey(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.SetPermissionDefinition(ctx, models.Permission{Permission: "run_poll", RoleKind: "chair"}); !errors.Is(err, permissions.ErrInvalidPartition) {
		t.Fatalf("no partition: %v", err)
	}
	if _, err := mgr.SetPermissionDefinition(ctx, models.Permission{Partition: "agy1", Permission: "run.*", RoleKind: "chair"}); !errors.Is(err, permissions.ErrInvalidPermission) {
		t.Fatalf("pattern permission: %v", err)
	}
}

func TestAPartitionsPermissionsListWithTheOverrideWinning(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	for _, p := range []models.Permission{
		{Partition: "agy1", Permission: "run_poll", RoleKind: "treasurer", Held: true},
		{Partition: "agy1", Permission: "run_poll", RoleKind: "chair", Held: true},
		{Partition: "agy1", Permission: "post_notice", RoleKind: "chair", Held: false},
		{Partition: "agy2", Permission: "run_poll", RoleKind: "chair", Held: true},
	} {
		if _, err := mgr.SetPermissionDefinition(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := mgr.OverridePermission(ctx, models.Permission{Partition: "agy1", Permission: "run_poll", RoleKind: "treasurer", Held: false, SetBy: "owner-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.OverridePermission(ctx, models.Permission{Partition: "agy1", Permission: "post_event", RoleKind: "secretary", Held: true, SetBy: "owner-1"}); err != nil {
		t.Fatal(err)
	}

	got, err := mgr.ListPermissions(ctx, "agy1")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		permission, kind string
		held             bool
		layer            models.PermissionLayer
		setBy            string
	}{
		{"post_event", "secretary", true, models.PermissionLayerOverride, "owner-1"},
		{"post_notice", "chair", false, models.PermissionLayerDefinition, ""},
		{"run_poll", "chair", true, models.PermissionLayerDefinition, ""},
		{"run_poll", "treasurer", false, models.PermissionLayerOverride, "owner-1"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d permissions, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Partition != "agy1" || g.Permission != w.permission || g.RoleKind != w.kind || g.Held != w.held || g.Layer != w.layer || g.SetBy != w.setBy || g.SetAt == "" {
			t.Fatalf("row %d: got %+v, want %+v", i, g, w)
		}
	}

	if err := mgr.ClearPermissionOverride(ctx, "agy1", "run_poll", "treasurer"); err != nil {
		t.Fatal(err)
	}
	got, _ = mgr.ListPermissions(ctx, "agy1")
	if g := got[3]; g.Layer != models.PermissionLayerDefinition || !g.Held {
		t.Fatalf("after clear: %+v", g)
	}
}

func TestAnEmptyPartitionListsNoPermissionsAndABadOneIsRefused(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	got, err := mgr.ListPermissions(ctx, "agy9")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty partition: %+v %v", got, err)
	}
	if _, err := mgr.ListPermissions(ctx, ""); !errors.Is(err, permissions.ErrInvalidPartition) {
		t.Fatalf("no partition: %v", err)
	}
}

func TestAHeldPermissionGrantsNoActionThroughTheGate(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	if _, err := mgr.CreateRole(ctx, models.Role{Code: "secretary", Name: "Secretary"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.CreateGrant(ctx, models.Grant{Subject: "ann", Scope: "module:engage", Partition: "agy1", RoleCode: "secretary"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.SetPermissionDefinition(ctx, models.Permission{Partition: "agy1", Permission: "post_notice", RoleKind: "secretary", Held: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.OverridePermission(ctx, models.Permission{Partition: "agy1", Permission: "post_event", RoleKind: "secretary", Held: true, SetBy: "owner-1"}); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"engage.notice.post", "engage.event.post"} {
		ok, err := mgr.MayPerformIn(ctx, "ann", "agy1", "module:engage", action)
		if err != nil || ok {
			t.Fatalf("MayPerformIn(%s) = %v, %v; a held permission must not reach the gate", action, ok, err)
		}
	}
	actions, err := mgr.ActionsFor(ctx, "ann", "module:engage")
	if err != nil || len(actions) != 0 {
		t.Fatalf("ActionsFor = %v, %v; want none from a held permission", actions, err)
	}
}
