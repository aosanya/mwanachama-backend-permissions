# mwanachama-backend-permissions — completed

Rows are cut here from [todo.md](todo.md) when they close.

| Task | Title | Closed | Notes |
|------|-------|--------|-------|
| PM0 | 🧱 **The module: blueprint, manager, generated routes.** | 2026-09-27 | Built on `mwanachama-backend-catalog`'s shape at the owner's instruction, after converting `mwanachama-backend-auth` was ruled out — Supabase holds authentication, and auth's storage needs five things the engine does not express (bytes, SQL NULL, a partial index on `retired_at`, Postgres sequences, `time.Time` columns). Three declared objects (`role`, `role_action`, `grant`), a `PermissionsManager` with `MayPerform` as one indexed three-way join, 14 operations generated into routes by `dispatch` and checked against the manager's real signatures at build, and two example domains that disagree about every noun. 16 tests: the gate refusing what a role does not list, scoping, the `*` scope, withdrawal, role retirement, malformed actions and scopes refused, reinstatement rather than duplication, and `SetActions` replacing a list atomically. `domain_agnostic_test.go` was mutation-checked — adding an identifier named `agencyScopePrefix` fails the build, which is the only evidence the rule is enforced rather than stated. |
