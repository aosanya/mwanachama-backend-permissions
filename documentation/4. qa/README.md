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

`routes/routes_security_test.go` drives the declared operations through
`dispatch`, the way a mount will. Since 2026-09-27 it is **red by design**:
four tests assert the fixed behaviour of open security rows and fail until
each is fixed — `TestPM5_…GrantsItselfEverywhere` (PM5, P1),
`TestPM6_…GrantedByIsTakenFromTheRequestBody` (PM6, P1),
`TestPM7_…SeededIsTakenFromTheRequestBody` (PM7, P3) and
`TestPM8_…GrantListingHasNoCap` (PM8, P3). Its fifth test,
`TestGrantWithASlashInItsScopeCanBeWithdrawnOverHTTP`, is a control that
holds and stays green. The root package is green.

Not covered yet: Postgres (`PM2`).
