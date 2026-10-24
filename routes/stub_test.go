package routes_test

import (
	"context"

	"github.com/aosanya/mwanachama-backend-permissions/models"
)

type stubManager struct{}

func (*stubManager) CreateRole(context.Context, models.Role) (models.Role, error) { panic("unused") }
func (*stubManager) GetRole(context.Context, string) (models.Role, error)         { panic("unused") }
func (*stubManager) ListRoles(context.Context, models.RoleFilter) ([]models.Role, error) {
	panic("unused")
}
func (*stubManager) UpdateRole(context.Context, models.Role) (models.Role, error) { panic("unused") }
func (*stubManager) DeleteRole(context.Context, string) error                     { panic("unused") }
func (*stubManager) AddAction(context.Context, string, string) (models.RoleAction, error) {
	panic("unused")
}
func (*stubManager) RemoveAction(context.Context, string, string) error { panic("unused") }
func (*stubManager) ListActions(context.Context, string) ([]models.RoleAction, error) {
	panic("unused")
}
func (*stubManager) SetActions(context.Context, string, []string) ([]models.RoleAction, error) {
	panic("unused")
}
func (*stubManager) CreateGrant(context.Context, models.Grant) (models.Grant, error) {
	panic("unused")
}
func (*stubManager) ListGrants(context.Context, models.GrantFilter) ([]models.Grant, error) {
	panic("unused")
}
func (*stubManager) WithdrawGrant(context.Context, string, string, string) error { panic("unused") }
func (*stubManager) WithdrawGrantIn(context.Context, string, string, string, string) error {
	panic("unused")
}
func (*stubManager) Seed(context.Context, string, string, ...[]byte) ([]models.RoleAction, error) {
	panic("unused")
}
func (*stubManager) MayPerform(context.Context, string, string, string) (bool, error) {
	panic("unused")
}
func (*stubManager) MayPerformIn(context.Context, string, string, string, string) (bool, error) {
	panic("unused")
}
func (*stubManager) ActionsFor(context.Context, string, string) ([]string, error) { panic("unused") }
func (*stubManager) ActionsAnywhere(context.Context, string) ([]string, error)    { panic("unused") }

func (*stubManager) SeedActions(context.Context, string, []string) ([]models.RoleAction, error) {
	panic("unused")
}
func (*stubManager) SetPowerDefinition(context.Context, models.Power) (models.Power, error) {
	panic("unused")
}
func (*stubManager) OverridePower(context.Context, models.Power) (models.Power, error) {
	panic("unused")
}
func (*stubManager) ClearPowerOverride(context.Context, string, string, string) error {
	panic("unused")
}
func (*stubManager) ReadPower(context.Context, string, string, string) (models.Power, error) {
	panic("unused")
}
