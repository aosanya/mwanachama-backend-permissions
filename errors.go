package mwanachamapermissions

import "errors"

var ErrInvalidRole = errors.New("permissions: a role needs a code and a name")

var ErrRoleNotFound = errors.New("permissions: no such role")

var ErrDuplicateRole = errors.New("permissions: that role code is taken")

var ErrInvalidAction = errors.New("permissions: that is not a well-formed action")

var ErrInvalidScope = errors.New("permissions: that is not a well-formed scope")

var ErrInvalidPartition = errors.New("permissions: that is not a well-formed partition")

var ErrInvalidGrant = errors.New("permissions: a grant needs a subject, a scope and a role")

var ErrGrantNotFound = errors.New("permissions: no such grant")

var ErrDuplicateGrant = errors.New("permissions: that subject already holds that role there")

var ErrNotPermitted = errors.New("permissions: the caller may not perform that action")

var ErrSeededRole = errors.New("permissions: a seeded role's actions are owned by the seeding pass")
