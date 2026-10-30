package mwanachamapermissions

import "errors"

var ErrInvalidRole = errors.New("permissions: a role needs a code and a name")

var ErrRoleNotFound = errors.New("permissions: no such role")

var ErrDuplicateRole = errors.New("permissions: that role code is taken")

var ErrInvalidAction = errors.New("permissions: that is not a well-formed action")

var ErrInvalidScope = errors.New("permissions: that is not a well-formed scope")

var ErrInvalidPartition = errors.New("permissions: that is not a well-formed partition")

var ErrInvalidGrant = errors.New("permissions: a grant needs a subject, a scope and a role")

var ErrInvalidEnd = errors.New("permissions: a grant's end must be an RFC 3339 time still to come")

var ErrGrantNotFound = errors.New("permissions: no such grant")

var ErrDuplicateGrant = errors.New("permissions: that subject already holds that role there")

var ErrNotPermitted = errors.New("permissions: the caller may not perform that action")

var ErrSeededRole = errors.New("permissions: a seeded role's actions are owned by the seeding pass")

var ErrInvalidPower = errors.New("permissions: a power setting needs a partition, a power code, a role kind and, for an override, who set it")

var ErrPowerNotFound = errors.New("permissions: no layer sets that power for that role kind")

var ErrPowersNotDeclared = errors.New("permissions: this domain declares no power object")

var ErrConflict = errors.New("permissions: another write got there first, try again")

var ErrActionsRequired = errors.New("permissions: replacing a role's actions needs the whole list, an empty one to clear it")

var ErrRoleRetired = errors.New("permissions: that role has been retired")
