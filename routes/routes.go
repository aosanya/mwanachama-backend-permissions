package routes

import (
	"fmt"
	"sync"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

type Route = httpwire.Route

var operations = sync.OnceValues(func() (*dispatch.Spec, error) {
	return dispatch.Parse(permissions.Operations())
})

var sentinels = map[string]error{
	"ErrInvalidRole":      permissions.ErrInvalidRole,
	"ErrRoleNotFound":     permissions.ErrRoleNotFound,
	"ErrDuplicateRole":    permissions.ErrDuplicateRole,
	"ErrInvalidAction":    permissions.ErrInvalidAction,
	"ErrInvalidScope":     permissions.ErrInvalidScope,
	"ErrInvalidPartition": permissions.ErrInvalidPartition,
	"ErrInvalidGrant":     permissions.ErrInvalidGrant,
	"ErrGrantNotFound":    permissions.ErrGrantNotFound,
	"ErrDuplicateGrant":   permissions.ErrDuplicateGrant,
	"ErrSeededRole":       permissions.ErrSeededRole,
	"ErrNotPermitted":     permissions.ErrNotPermitted,
}

var AnonymousActions []string

type Mount struct {
	Caller dispatch.Caller
}

func Build(pm permissions.PermissionsManager) ([]Route, error) { return BuildFor(pm, Mount{}) }

func BuildFor(pm permissions.PermissionsManager, m Mount) ([]Route, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	return dispatch.Dispatch(s, dispatch.Deps{Manager: pm, Errors: sentinels, Caller: m.Caller})
}

func Routes(pm permissions.PermissionsManager) []Route { return RoutesFor(pm, Mount{}) }

func RoutesFor(pm permissions.PermissionsManager, m Mount) []Route {
	out, err := BuildFor(pm, m)
	if err != nil {
		panic(fmt.Sprintf("permissions routes: %v", err))
	}
	return out
}
