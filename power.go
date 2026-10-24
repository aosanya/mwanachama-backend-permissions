package mwanachamapermissions

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func (m *manager) SetPowerDefinition(ctx context.Context, p models.Power) (models.Power, error) {
	p.Layer, p.SetBy = models.PowerLayerDefinition, ""
	return m.putPower(ctx, p)
}

func (m *manager) OverridePower(ctx context.Context, p models.Power) (models.Power, error) {
	if p.SetBy == "" {
		return models.Power{}, ErrInvalidPower
	}
	p.Layer = models.PowerLayerOverride
	return m.putPower(ctx, p)
}

func (m *manager) ClearPowerOverride(ctx context.Context, partition, power, roleKind string) error {
	if !m.powersDeclared() {
		return ErrPowersNotDeclared
	}
	if err := checkPowerKey(partition, power, roleKind); err != nil {
		return err
	}
	err := m.query(ctx, rolePower).
		Where(keyOfPower(), partition, power, roleKind, models.PowerLayerOverride).
		Delete(map[string]any{}).Error
	if err != nil {
		return fmt.Errorf("permissions: clear power override: %w", err)
	}
	return nil
}

func (m *manager) ReadPower(ctx context.Context, partition, power, roleKind string) (models.Power, error) {
	if !m.powersDeclared() {
		return models.Power{}, ErrPowersNotDeclared
	}
	if err := checkPowerKey(partition, power, roleKind); err != nil {
		return models.Power{}, err
	}
	for _, layer := range []models.PowerLayer{models.PowerLayerOverride, models.PowerLayerDefinition} {
		p, err := m.findPower(ctx, partition, power, roleKind, layer)
		if err == nil {
			return p, nil
		}
		if !errors.Is(err, ErrPowerNotFound) {
			return models.Power{}, err
		}
	}
	return models.Power{}, ErrPowerNotFound
}

func (m *manager) putPower(ctx context.Context, p models.Power) (models.Power, error) {
	if !m.powersDeclared() {
		return models.Power{}, ErrPowersNotDeclared
	}
	if err := checkPowerKey(p.Partition, p.Power, p.RoleKind); err != nil {
		return models.Power{}, err
	}
	p.SetAt = models.NowRFC3339()
	row, err := encode(m.object(rolePower), p)
	if err != nil {
		return models.Power{}, err
	}
	err = m.query(ctx, rolePower).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: columnName("Partition")}, {Name: columnName("Power")},
			{Name: columnName("RoleKind")}, {Name: columnName("Layer")},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			columnName("Held"), columnName("SetBy"), columnName("SetAt"),
		}),
	}).Create(row).Error
	if err != nil {
		return models.Power{}, fmt.Errorf("permissions: set power: %w", err)
	}
	return p, nil
}

func (m *manager) findPower(ctx context.Context, partition, power, roleKind string, layer models.PowerLayer) (models.Power, error) {
	var row map[string]any
	err := m.query(ctx, rolePower).
		Where(keyOfPower(), partition, power, roleKind, layer).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Power{}, ErrPowerNotFound
	}
	if err != nil {
		return models.Power{}, fmt.Errorf("permissions: read power: %w", err)
	}
	var out models.Power
	if err := decode(m.object(rolePower), row, &out); err != nil {
		return models.Power{}, err
	}
	return out, nil
}

func keyOfPower() string {
	return columnName("Partition") + " = ? and " +
		columnName("Power") + " = ? and " +
		columnName("RoleKind") + " = ? and " +
		columnName("Layer") + " = ?"
}

func checkPowerKey(partition, power, roleKind string) error {
	if partition == "" || !isPartition(partition) {
		return ErrInvalidPartition
	}
	if !isCode(power) || !isCode(roleKind) {
		return ErrInvalidPower
	}
	return nil
}

func (m *manager) powersDeclared() bool {
	_, ok := m.spec.ByRole(rolePower)
	return ok
}

func (m *manager) ListPowers(ctx context.Context, partition string) ([]models.Power, error) {
	if !m.powersDeclared() {
		return nil, ErrPowersNotDeclared
	}
	if partition == "" || !isPartition(partition) {
		return nil, ErrInvalidPartition
	}
	var rows []map[string]any
	err := m.query(ctx, rolePower).
		Where(columnName("Partition")+" = ?", partition).
		Order(columnName("Power") + ", " + columnName("RoleKind")).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("permissions: list powers: %w", err)
	}
	out := []models.Power{}
	at := map[[2]string]int{}
	for _, row := range rows {
		var p models.Power
		if err := decode(m.object(rolePower), row, &p); err != nil {
			return nil, err
		}
		key := [2]string{p.Power, p.RoleKind}
		i, seen := at[key]
		if !seen {
			at[key] = len(out)
			out = append(out, p)
			continue
		}
		if p.Layer == models.PowerLayerOverride {
			out[i] = p
		}
	}
	return out, nil
}
