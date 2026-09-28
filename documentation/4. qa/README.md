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

`routes/withdraw_subject_security_test.go` (2026-09-28, eighteenth sweep)
pins the same for the subject: a grant to `team/alice` is withdrawn through
`/grants/team%2Falice/…`, and a subject that is itself the literal text
`a%2Fb` through `/grants/a%252Fb/…` — neither is left holding a grant no
route can reach. `routes/repeated_query_security_test.go` pins that a
repeated `subject` on `GET /check` and `GET /grants` is read the same way by
both (the first value), so a screen listing grants and the check that
enforces them cannot disagree about who was asked for.

`routes/action_security_test.go` pins two more refusals through the route:
a role cannot be given a wildcard (`catalog.*`, `*`), miscased or empty
action — the "no wildcards in `role_action`" rule, which `SetActions` was
already tested for but `AddAction` was not — and a grant made to a padded
subject (`" bob "`) answers `false` for the trimmed one, so a subject string
fails closed rather than normalising into someone else's grants.

`routes/lifecycle_security_test.go` (2026-09-27, ninth sweep) pins the grant
lifecycle through the route: deleting a role stops every grant of it
authorizing at once, and recreating the same code is refused (409) rather
than reviving the old grants; a grant in `instance:a` authorizes nothing in
`instance:A`, `instance:a/catalog`, `instance` or `instance:ab`; a
`"deleted": true` in the create body is ignored; a withdrawn grant stops on
the next check.

`routes/update_role_security_test.go` (2026-09-28, fifteenth sweep) pins that
`PATCH /roles/writer` with a body naming `"code":"admin"` renames `writer` and
leaves `admin` alone — the path outranks the body — and that the body cannot
set `seeded` or `deleted`. `routes/grant_reinstate_security_test.go` pins that
a withdrawn grant reinstated by a second caller records that caller as
`granted_by`, not the body's claim and not the original grantor — the
`reinstate` path writes the column separately from `CreateGrant`'s insert.

`seed_retired_security_test.go` (2026-09-28, twentieth sweep) pins what
`Seed` does to a role somebody retired. `mwanachama-wakala-api`'s
`cmd/seedrole` re-seeds `studio_operator` "whenever a module joins that
list", so a deploy can run `Seed` over a role an operator retired to revoke
everyone who held it. `Seed` finds the retired row, skips `CreateRole`, and
rewrites the role's actions — but the role stays `deleted`, and its standing
`*` grant still permits nothing (`MayPerform` false for every seeded action,
`ActionsAnywhere` empty). Mutation-checked: dropping the retired-role
predicate from `MayPerform`'s join turns it red.

## Red by design

- `routes/role_seeded_security_test.go`'s
  `TestPM6_OpenHole_CreateRoleTakesSeededFromTheRequestBody` (PM6, P2): a
  role declared with `"seeded": true` in the body is stored seeded, and
  `AddAction` on it then answers 409. Leave it red until `CreateRole` stops
  taking `seeded` from the request.

PM5, the red this section carried before, was cleared 2026-09-27 and
its test kept as the regression guard it became:
`routes/grant_security_test.go`'s
`TestPM5_GrantedByIsNeverTakenFromTheRequestBody`.

Three tests hold the grantor now, and the second two were added with the fix:

- `TestPM5_GrantedByIsNeverTakenFromTheRequestBody` — the stored `granted_by`
  is never the value the body claimed.
- `TestPM5_TheMountSuppliesTheGrantor` — mounted through
  `routes.Mount{Caller: …}`, the stored `granted_by` **is** the caller the
  mount supplied, so the fix records the trail rather than emptying it.
- `TestPM5_AGrantorArgumentIsNeverOfferedToAnAgent` — `create_grant` binds
  `GrantedBy` only `from: "caller"`, read off the shipped spec, so a later
  edit cannot quietly move it back to the body.

`create_grant` declares
`{"from": "caller", "as": "granted_by", "into": "GrantedBy"}`, a source
`mwanachama-backend-shared`'s `dispatch` gained for catalog's CAT12 and which
was extended to whole-body overwrites for this row (S23 there). A mount that
supplies no `Caller` records `""` — never the request's value — which is what
`routes.Routes(pm)` still does, since nothing mounts these routes yet.
