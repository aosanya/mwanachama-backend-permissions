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
