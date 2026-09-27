# mwanachama-backend-permissions

Who may perform which declared action, where.

A Go library backed by Postgres or SQLite through GORM, with its objects
declared rather than written — `Migrate(db, spec)` creates the tables and
`NewPermissionsManager(db, spec)` returns the manager. There is no service to
run.

## A permission is a declared action

Every operation in a spec-driven module already carries one, because
`mwanachama-backend-shared/dispatch` requires it:

```
agency.goal.create      catalog.entry.upsert      permissions.grant.create
```

This module stores which of them a bundle permits and who holds that bundle
where. It never learns what any of them do.

## Three objects

| role | holds | key |
| --- | --- | --- |
| `role` | a named bundle | `code` |
| `role_action` | one action it permits | `role_code` + `action` |
| `grant` | one subject holding it in one scope | `subject` + `scope` + `role_code` |

```go
mgr.CreateRole(ctx, permissions.Role{Code: "viewer", Name: "Viewer"})
mgr.AddAction(ctx, "viewer", "agency.module.list")
mgr.CreateGrant(ctx, permissions.Grant{Subject: uid, Scope: "instance:" + id, RoleCode: "viewer"})

ok, err := mgr.MayPerform(ctx, uid, "instance:"+id, "agency.module.list") // true
ok, err = mgr.MayPerform(ctx, uid, "instance:"+id, "agency.goal.create")  // false
```

## It names no domain

The same store holds a studio's operator roles and a hospital's clinical
privileges — `spec/examples/` carries both, and `domain_agnostic_test.go`
fails the build on a word that means something in only one of them.

```
wakala.permissions.json    role → "role"           → agency_permissions_roles
hospital.permissions.json  role → "privilege_set"  → stmarys_permissions_privilege_sets
```

Reasoning is in [documentation/2. design/permissions.md](documentation/2.%20design/permissions.md);
the cross-repo record is `developer`'s `architecture-permissions.md`.
