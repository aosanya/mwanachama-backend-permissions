# Drift between this module and the hub's in-tree copy

The hub carries a copy of this module at `code/wakala-api/modules/permissions`. This page records, as of 2026-11-19, every file and behaviour that exists on only one side or that differs beyond the import path, and what was done with each. It is the ground for replacing the in-tree copy with an import of this repository.

## How the two sides were compared

This repository has no `scripts/hubdiff` and no `hub-copy-diff.md` of its own, unlike its siblings. The comparison followed the siblings' recipe by hand:

1. Copy every `*.go` and `*.json` file of each side into a scratch directory.
2. Rewrite the hub copy's imports: `wakalaserver/modules/shared` to `github.com/aosanya/mwanachama-backend-shared`, and `wakalaserver/modules/permissions` to `github.com/aosanya/mwanachama-backend-permissions`. The package name is `mwanachamapermissions` on both sides already.
3. Normalise both trees with the `hubdiff` program (parse, drop every comment, reprint, `gofmt`).
4. `diff -r` the two trees.

## Result in one line

The non-test code, `models/`, the blueprint and the operations file are identical on both sides once comments are dropped. All the drift is in tests, in what only this repository carries, and in wiring that lives in the hub outside the module.

## Differences, with the side and the disposition

### Only in the hub

| Item | What it is | Disposition |
|---|---|---|
| `author_test.go` | Pins that a grant's grantor and a permission override's author come from the caller, never from the request body, and that the permission tools do not ask for an author. Internal-package test over the dispatch table. | Moved here unchanged apart from the import path. Passes. |
| `grants_access_cover_test.go` | Grant creation refusals, duplicate and retired-role cases, `MayPerform` scope and partition rules, `ListGrants` filters and order, `WithdrawGrantIn`, `ActionsFor`, `AccessOf`. | Moved here. Passes. |
| `permission_settings_cover_test.go` | Permission read and override layering, key refusals, `ListPermissions` merge, a domain that declares no permissions, scope and pattern shapes, conflict classification. Reaches unexported functions, so it stays in the internal package. | Moved here. Passes. |
| `roles_actions_cover_test.go` | Role create, update, list and delete refusals, `AddAction`, `SetActions`, `SeedActions`, `Seed`, `DeclaredActions`. | Moved here. Passes. |
| `sqlite_fixture_cover_test.go` | The `coverManager` fixture and the two in-line cover specs the four files above use, on in-memory SQLite. | Moved here. Passes. |
| `cmd/server/permissions.go` (hub) | Reads `PERMISSIONS_DATABASE_URL`, resolves the spec through the hub's `specMount`, opens Postgres, migrates and builds the manager. | Hub wiring, not part of the module. Not moved. |
| `internal/api/http/caches.go` (hub) | `CachePermissions`, a caching wrapper over `PermissionsManager` that drops cached answers when a grant, role, action or permission changes. | Hub wiring. Not moved. |
| `internal/api/http/deps.go`, `me_routes.go`, `resident_grants.go`, `record_grants.go` (hub) | The hub's own HTTP layer that holds a `PermissionsManager` and answers its screens. | Hub wiring. Not moved. |
| The hub's permission catalogue and tool filters | Which actions the hub declares, which roles hold them and which MCP tools each caller sees. | Hub wiring. Not moved. |

### Only in this repository

| Item | What it is | Disposition |
|---|---|---|
| Test files at the root: `access_test.go`, `access_withdrawn_order_test.go`, `action_race_test.go`, `action_retired_role_test.go`, `action_setactions_test.go`, `collect_grant_test.go`, `delete_seeded_role_test.go`, `domain_agnostic_test.go`, `grant_key_security_test.go`, `grant_race_test.go`, `grant_retired_role_test.go`, `like_filter_test.go`, `list_cap_test.go`, `list_order_test.go`, `listing_matches_gate_security_test.go`, `partition_test.go`, `permission_rename_test.go`, `permission_test.go`, `permissions_test.go`, `record_scope_test.go`, `role_code_revival_security_test.go`, `role_race_test.go`, `scope_match_security_test.go`, `seed_retired_security_test.go`, `seed_test.go`, `subject_star_security_test.go`, `testdb_test.go` | The module's own suite: security regressions, race cases, the domain-agnostic guard, listing caps and order. | Kept. The hub copy never had them, so the hub copy is the poorer side. |
| `routes/` (`routes.go` and 37 test files) | The route table and MCP tools built from the operations file, with the security regressions over the HTTP surface. | Kept. The hub does not build this table from the module; it mounts permissions through its own layer. Nothing to merge. |
| `spec/examples/hospital.permissions.json`, `spec/examples/wakala.permissions.json` | Two example domain specs that disagree about every noun, loaded by the tests. | Kept. |
| `CLAUDE.md`, `README.md`, `LICENSE`, `documentation/`, `go.mod`, `go.sum` | Repository files. | Kept. |

### Present on both sides but different beyond the import path

Nothing. After the import rewrite and comment stripping, these are byte-identical: `access.go`, `action.go`, `blueprint.go`, `conflict.go`, `errors.go`, `grant.go`, `like.go`, `list_cap.go`, `partition.go`, `patterns.go`, `permission.go`, `permission_rename.go`, `permissions_manager.go`, `role.go`, `seed.go`, `store.go`, `models/*.go`, `permissions.blueprint.json`, `permissions.operations.json`.

The one raw difference is `doc.go`, where this repository's package comment adds the line `See documentation/2. design/permissions.md.` Only a comment differs.

## Behaviour conflicts

None. The code is the same on both sides, and the five hub test files pass against this repository's code, so no operation behaves differently on the two sides.

## What changed here as a result

- The five hub-only test files were added at the root, with the import path rewritten and no other edit.
- No source file, blueprint, operations file or pinned count changed.

## Known failure not caused by this module

`go test ./...` in this repository passes at the root and fails two tests in `routes/`:

- `TestAQuestionNamingNoScopeIsRefusedEvenForAnEverywhereGrant`
- `TestAnEmptySubjectCanNeitherHoldAGrantNorPassTheCheck`

Both ask the check route for a missing `scope` or `subject` and expect `200 false`. They get `400` with `the query parameter is required`, because `mwanachama-backend-shared/dispatch/bind.go` now refuses a missing required query argument (added in that repository on 2026-11-16, after these tests were written). The module resolves shared through a local `replace` to `../mwanachama-backend-shared`. The failure is the same before and after the move and is for the shared module's owner to settle: either the two tests change to expect `400`, or the check route's arguments stop being declared required.
