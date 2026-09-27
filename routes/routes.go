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
	"ErrInvalidRole":    permissions.ErrInvalidRole,
	"ErrRoleNotFound":   permissions.ErrRoleNotFound,
	"ErrDuplicateRole":  permissions.ErrDuplicateRole,
	"ErrInvalidAction":  permissions.ErrInvalidAction,
	"ErrInvalidScope":   permissions.ErrInvalidScope,
	"ErrInvalidGrant":   permissions.ErrInvalidGrant,
	"ErrGrantNotFound":  permissions.ErrGrantNotFound,
	"ErrDuplicateGrant": permissions.ErrDuplicateGrant,
	"ErrSeededRole":     permissions.ErrSeededRole,
	"ErrNotPermitted":   permissions.ErrNotPermitted,
}

var AnonymousActions []string

func Build(pm permissions.PermissionsManager) ([]Route, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	return dispatch.Dispatch(s, dispatch.Deps{Manager: pm, Errors: sentinels})
}

func Routes(pm permissions.PermissionsManager) []Route {
	out, err := Build(pm)
	if err != nil {
		panic(fmt.Sprintf("permissions routes: %v", err))
	}
	return out
}
