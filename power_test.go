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
	key := models.Power{Partition: "agy1", Power: "run_poll", RoleKind: "treasurer"}

	def := key
	def.Held = true
	if _, err := mgr.SetPowerDefinition(ctx, def); err != nil {
		t.Fatalf("definition: %v", err)
	}
	got, err := mgr.ReadPower(ctx, "agy1", "run_poll", "treasurer")
	if err != nil || !got.Held || got.Layer != models.PowerLayerDefinition || got.SetBy != "" {
		t.Fatalf("read definition: %+v %v", got, err)
	}

	over := key
	over.Held, over.SetBy = false, "owner-1"
	if _, err := mgr.OverridePower(ctx, over); err != nil {
		t.Fatalf("override: %v", err)
	}
	got, err = mgr.ReadPower(ctx, "agy1", "run_poll", "treasurer")
	if err != nil || got.Held || got.Layer != models.PowerLayerOverride || got.SetBy != "owner-1" || got.SetAt == "" {
		t.Fatalf("read override: %+v %v", got, err)
	}

	def.Held = false
	if _, err := mgr.SetPowerDefinition(ctx, def); err != nil {
		t.Fatalf("republish: %v", err)
	}
	if got, _ := mgr.ReadPower(ctx, "agy1", "run_poll", "treasurer"); got.Layer != models.PowerLayerOverride {
		t.Fatalf("a republish displaced the override: %+v", got)
	}

	if err := mgr.ClearPowerOverride(ctx, "agy1", "run_poll", "treasurer"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, err = mgr.ReadPower(ctx, "agy1", "run_poll", "treasurer")
	if err != nil || got.Layer != models.PowerLayerDefinition || got.Held {
		t.Fatalf("after clear: %+v %v", got, err)
	}
}

func TestAPowerIsHeldPerPartitionAndRoleKind(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	if _, err := mgr.SetPowerDefinition(ctx, models.Power{Partition: "agy1", Power: "post_notice", RoleKind: "secretary", Held: true}); err != nil {
		t.Fatal(err)
	}
	for _, k := range [][3]string{{"agy2", "post_notice", "secretary"}, {"agy1", "post_notice", "chair"}, {"agy1", "post_event", "secretary"}} {
		if _, err := mgr.ReadPower(ctx, k[0], k[1], k[2]); !errors.Is(err, permissions.ErrPowerNotFound) {
			t.Fatalf("%v: want ErrPowerNotFound, got %v", k, err)
		}
	}
}

func TestAnOverrideNeedsWhoSetIt(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.OverridePower(context.Background(), models.Power{Partition: "agy1", Power: "run_poll", RoleKind: "chair", Held: true})
	if !errors.Is(err, permissions.ErrInvalidPower) {
		t.Fatalf("want ErrInvalidPower, got %v", err)
	}
}

func TestAPowerSettingRefusesAMalformedKey(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.SetPowerDefinition(ctx, models.Power{Power: "run_poll", RoleKind: "chair"}); !errors.Is(err, permissions.ErrInvalidPartition) {
		t.Fatalf("no partition: %v", err)
	}
	if _, err := mgr.SetPowerDefinition(ctx, models.Power{Partition: "agy1", Power: "run.*", RoleKind: "chair"}); !errors.Is(err, permissions.ErrInvalidPower) {
		t.Fatalf("pattern power: %v", err)
	}
}

func TestAPartitionsPowersListWithTheOverrideWinning(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	for _, p := range []models.Power{
		{Partition: "agy1", Power: "run_poll", RoleKind: "treasurer", Held: true},
		{Partition: "agy1", Power: "run_poll", RoleKind: "chair", Held: true},
		{Partition: "agy1", Power: "post_notice", RoleKind: "chair", Held: false},
		{Partition: "agy2", Power: "run_poll", RoleKind: "chair", Held: true},
	} {
		if _, err := mgr.SetPowerDefinition(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := mgr.OverridePower(ctx, models.Power{Partition: "agy1", Power: "run_poll", RoleKind: "treasurer", Held: false, SetBy: "owner-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.OverridePower(ctx, models.Power{Partition: "agy1", Power: "post_event", RoleKind: "secretary", Held: true, SetBy: "owner-1"}); err != nil {
		t.Fatal(err)
	}

	got, err := mgr.ListPowers(ctx, "agy1")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		power, kind string
		held        bool
		layer       models.PowerLayer
		setBy       string
	}{
		{"post_event", "secretary", true, models.PowerLayerOverride, "owner-1"},
		{"post_notice", "chair", false, models.PowerLayerDefinition, ""},
		{"run_poll", "chair", true, models.PowerLayerDefinition, ""},
		{"run_poll", "treasurer", false, models.PowerLayerOverride, "owner-1"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d powers, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Partition != "agy1" || g.Power != w.power || g.RoleKind != w.kind || g.Held != w.held || g.Layer != w.layer || g.SetBy != w.setBy || g.SetAt == "" {
			t.Fatalf("row %d: got %+v, want %+v", i, g, w)
		}
	}

	if err := mgr.ClearPowerOverride(ctx, "agy1", "run_poll", "treasurer"); err != nil {
		t.Fatal(err)
	}
	got, _ = mgr.ListPowers(ctx, "agy1")
	if g := got[3]; g.Layer != models.PowerLayerDefinition || !g.Held {
		t.Fatalf("after clear: %+v", g)
	}
}

func TestAnEmptyPartitionListsNoPowersAndABadOneIsRefused(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	got, err := mgr.ListPowers(ctx, "agy9")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty partition: %+v %v", got, err)
	}
	if _, err := mgr.ListPowers(ctx, ""); !errors.Is(err, permissions.ErrInvalidPartition) {
		t.Fatalf("no partition: %v", err)
	}
}

func TestAHeldPowerGrantsNoActionThroughTheGate(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	if _, err := mgr.CreateRole(ctx, models.Role{Code: "secretary", Name: "Secretary"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.CreateGrant(ctx, models.Grant{Subject: "ann", Scope: "module:engage", Partition: "agy1", RoleCode: "secretary"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.SetPowerDefinition(ctx, models.Power{Partition: "agy1", Power: "post_notice", RoleKind: "secretary", Held: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.OverridePower(ctx, models.Power{Partition: "agy1", Power: "post_event", RoleKind: "secretary", Held: true, SetBy: "owner-1"}); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"engage.notice.post", "engage.event.post"} {
		ok, err := mgr.MayPerformIn(ctx, "ann", "agy1", "module:engage", action)
		if err != nil || ok {
			t.Fatalf("MayPerformIn(%s) = %v, %v; a held power must not reach the gate", action, ok, err)
		}
	}
	actions, err := mgr.ActionsFor(ctx, "ann", "module:engage")
	if err != nil || len(actions) != 0 {
		t.Fatalf("ActionsFor = %v, %v; want none from a held power", actions, err)
	}
}
