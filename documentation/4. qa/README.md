# QA

`go test ./...` runs everything on SQLite in memory.

Two suites carry most of the weight:

- `permissions_test.go` — the gate. What a role does not list is refused,
  scopes do not leak into each other, `*` reaches everywhere, and a withdrawn
  grant or a retired role permits nothing.
- `domain_agnostic_test.go` — the rule that this module names no domain,
  enforced by parsing the package's own identifiers and its blueprint. It is
  mutation-checked: an identifier carrying a domain word fails the build.

`routes/routes_test.go` builds the declared operations against the manager
interface, which is where a renamed method or a mistyped argument is caught —
at build, not at the first request.

Not covered yet: Postgres (`PM2`).

`routes/withdraw_scope_security_test.go` pins that a grant whose scope
carries a slash (`instance:abc/catalog`) can still be withdrawn through the
route, with the slash sent as `%2F`, and permits nothing afterwards.

`routes/action_security_test.go` pins two more refusals through the route:
a role cannot be given a wildcard (`catalog.*`, `*`), miscased or empty
action — the "no wildcards in `role_action`" rule, which `SetActions` was
already tested for but `AddAction` was not — and a grant made to a padded
subject (`" bob "`) answers `false` for the trimmed one, so a subject string
fails closed rather than normalising into someone else's grants.

## Red by design

`routes/grant_security_test.go`'s
`TestPM5_OpenHole_GrantedByIsTakenFromTheRequestBody` fails on purpose: it
asserts that a grant's `granted_by` is never the value the request body
claimed, and today it is (`PM5` on the board). It turns green when the fix
lands; do not delete or invert it to get `go test ./...` green.
