package mwanachamapermissions

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

func (m *manager) CreateGrant(ctx context.Context, g models.Grant) (models.Grant, error) {
	if g.Subject == "" || g.Scope == "" || g.RoleCode == "" {
		return models.Grant{}, ErrInvalidGrant
	}
	if !isScope(g.Scope) {
		return models.Grant{}, ErrInvalidScope
	}
	if _, err := m.findRole(ctx, g.RoleCode); err != nil {
		return models.Grant{}, err
	}

	existing, err := m.findGrant(ctx, g.Subject, g.Scope, g.RoleCode)
	switch {
	case err == nil && !existing.Deleted:
		return models.Grant{}, ErrDuplicateGrant
	case err == nil:
		return m.reinstate(ctx, g)
	case !errors.Is(err, ErrGrantNotFound):
		return models.Grant{}, err
	}

	g.GrantedAt, g.Deleted = models.NowRFC3339(), false
	row, err := encode(m.object(roleGrant), g)
	if err != nil {
		return models.Grant{}, err
	}
	if err := m.query(ctx, roleGrant).Create(row).Error; err != nil {
		return models.Grant{}, fmt.Errorf("permissions: create grant: %w", err)
	}
	return g, nil
}

// reinstate turns a withdrawn grant back on rather than inserting a second
// row, because the three keys are the whole primary key. The trail is what
// granted_at and granted_by say now, not a row per decision — a fuller audit
// belongs in whatever records the act, not in the table the gate reads.
func (m *manager) reinstate(ctx context.Context, g models.Grant) (models.Grant, error) {
	g.GrantedAt, g.Deleted = models.NowRFC3339(), false
	err := m.query(ctx, roleGrant).
		Where(keyOfGrant(), g.Subject, g.Scope, g.RoleCode).
		Updates(map[string]any{
			columnName("Deleted"):   false,
			columnName("GrantedAt"): g.GrantedAt,
			columnName("GrantedBy"): g.GrantedBy,
		}).Error
	if err != nil {
		return models.Grant{}, fmt.Errorf("permissions: reinstate grant: %w", err)
	}
	return g, nil
}

func keyOfGrant() string {
	return columnName("Subject") + " = ? and " +
		columnName("Scope") + " = ? and " +
		columnName("RoleCode") + " = ?"
}

func (m *manager) findGrant(ctx context.Context, subject, scope, roleCode string) (models.Grant, error) {
	var row map[string]any
	err := m.query(ctx, roleGrant).
		Where(keyOfGrant(), subject, scope, roleCode).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Grant{}, ErrGrantNotFound
	}
	if err != nil {
		return models.Grant{}, fmt.Errorf("permissions: read grant: %w", err)
	}
	var out models.Grant
	if err := decode(m.object(roleGrant), row, &out); err != nil {
		return models.Grant{}, err
	}
	return out, nil
}

func (m *manager) ListGrants(ctx context.Context, f models.GrantFilter) ([]models.Grant, error) {
	q := m.query(ctx, roleGrant).Where(columnName("Deleted") + " = false")
	if f.Subject != "" {
		q = q.Where(columnName("Subject")+" = ?", f.Subject)
	}
	if f.Scope != "" {
		q = q.Where(columnName("Scope")+" = ?", f.Scope)
	}
	if f.RoleCode != "" {
		q = q.Where(columnName("RoleCode")+" = ?", f.RoleCode)
	}

	var rows []map[string]any
	err := q.Order(columnName("Subject") + ", " + columnName("Scope") + ", " + columnName("RoleCode")).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("permissions: list grants: %w", err)
	}
	out := make([]models.Grant, 0, len(rows))
	for _, row := range rows {
		var g models.Grant
		if err := decode(m.object(roleGrant), row, &g); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func (m *manager) WithdrawGrant(ctx context.Context, subject, scope, roleCode string) error {
	g, err := m.findGrant(ctx, subject, scope, roleCode)
	if err != nil {
		return err
	}
	if g.Deleted {
		return nil
	}
	err = m.query(ctx, roleGrant).
		Where(keyOfGrant(), subject, scope, roleCode).
		Update(columnName("Deleted"), true).Error
	if err != nil {
		return fmt.Errorf("permissions: withdraw grant: %w", err)
	}
	return nil
}
