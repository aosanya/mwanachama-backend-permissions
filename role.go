package mwanachamapermissions

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func (m *manager) CreateRole(ctx context.Context, r models.Role) (models.Role, error) {
	if !isCode(r.Code) || r.Name == "" {
		return models.Role{}, ErrInvalidRole
	}
	if _, err := m.findRole(ctx, r.Code); err == nil {
		return models.Role{}, ErrDuplicateRole
	} else if !errors.Is(err, ErrRoleNotFound) {
		return models.Role{}, err
	}

	now := models.NowRFC3339()
	r.CreatedAt, r.UpdatedAt, r.Deleted = now, now, false

	row, err := encode(m.object(roleRole), r)
	if err != nil {
		return models.Role{}, err
	}
	if err := m.query(ctx, roleRole).Create(row).Error; err != nil {
		return models.Role{}, fmt.Errorf("permissions: create role: %w", err)
	}
	return r, nil
}

func (m *manager) GetRole(ctx context.Context, code string) (models.Role, error) {
	return m.findRole(ctx, code)
}

func (m *manager) findRole(ctx context.Context, code string) (models.Role, error) {
	var row map[string]any
	err := m.query(ctx, roleRole).
		Where(columnName("Code")+" = ?", code).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Role{}, ErrRoleNotFound
	}
	if err != nil {
		return models.Role{}, fmt.Errorf("permissions: read role: %w", err)
	}
	var out models.Role
	if err := decode(m.object(roleRole), row, &out); err != nil {
		return models.Role{}, err
	}
	return out, nil
}

func (m *manager) ListRoles(ctx context.Context, f models.RoleFilter) ([]models.Role, error) {
	q := m.query(ctx, roleRole).Where(columnName("Deleted") + " = false")
	if f.Code != "" {
		q = q.Where(columnName("Code")+" = ?", f.Code)
	}
	if f.Name != "" {
		q = q.Where("lower("+columnName("Name")+") like ?", "%"+lower(f.Name)+"%")
	}
	if f.Seeded != nil {
		q = q.Where(columnName("Seeded")+" = ?", *f.Seeded)
	}

	var rows []map[string]any
	if err := q.Order(columnName("Code")).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("permissions: list roles: %w", err)
	}
	out := make([]models.Role, 0, len(rows))
	for _, row := range rows {
		var r models.Role
		if err := decode(m.object(roleRole), row, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (m *manager) UpdateRole(ctx context.Context, r models.Role) (models.Role, error) {
	existing, err := m.findRole(ctx, r.Code)
	if err != nil {
		return models.Role{}, err
	}
	if r.Name == "" {
		return models.Role{}, ErrInvalidRole
	}

	existing.Name, existing.Description = r.Name, r.Description
	existing.UpdatedAt = models.NowRFC3339()

	err = m.query(ctx, roleRole).
		Where(columnName("Code")+" = ?", r.Code).
		Updates(map[string]any{
			columnName("Name"):        existing.Name,
			columnName("Description"): existing.Description,
			columnName("UpdatedAt"):   existing.UpdatedAt,
		}).Error
	if err != nil {
		return models.Role{}, fmt.Errorf("permissions: update role: %w", err)
	}
	return existing, nil
}

func (m *manager) DeleteRole(ctx context.Context, code string) error {
	if _, err := m.findRole(ctx, code); err != nil {
		return err
	}
	err := m.query(ctx, roleRole).
		Where(columnName("Code")+" = ?", code).
		Updates(map[string]any{
			columnName("Deleted"):   true,
			columnName("UpdatedAt"): models.NowRFC3339(),
		}).Error
	if err != nil {
		return fmt.Errorf("permissions: delete role: %w", err)
	}
	return nil
}
