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

`routes/removed_action_security_test.go` (2026-10-02, sixty-third sweep) pins
the other half of a role's lifecycle: an action taken off a role stops
authorizing through every grant already standing on it, at once. `DELETE
/roles/editor/actions/catalog.entry.delete` turns erin's `/check` for that
action from `true` to `false` while the role's other action stays `true`, and a
`PUT /roles/curator/actions` that narrows the list does the same for finn's
`*` grant. Mutation-checked: pointing `RemoveAction`'s delete at a different
action turns the first test red.

`routes/update_role_security_test.go` (2026-09-28, fifteenth sweep) pins that
`PATCH /roles/writer` with a body naming `"code":"admin"` renames `writer` and
leaves `admin` alone — the path outranks the body — and that the body cannot
set `seeded` or `deleted`. `routes/rename_seeded_security_test.go`
(2026-10-01, forty-fifth sweep) pins the other direction: a `PATCH` whose
body says `"seeded":false` leaves a role a seeding pass owns sealed, and the
next `POST …/actions` on it still answers 409.
`routes/grant_reinstate_security_test.go` pins that
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

`scope_match_security_test.go` (2026-10-03, sixty-sixth sweep) pins that
`MayPerform` matches a grant by equality and nothing looser: a grant to
`member-a` on `instance:alpha` for `example.thing.update` authorizes no
longer, shorter, upper-cased, slash- or colon-extended scope, no `%`/`_`
wildcard scope or subject, no near-miss action, and not `*`; a
case-variant action is `ErrInvalidAction`. Its second test pins that a
retired role cannot come back under the same code: `CreateRole` answers
`ErrDuplicateRole`, a rename and an added action leave it retired, and its
standing grant still permits nothing. Mutation-checked: matching the scope
as a prefix `LIKE` turns four scope cases red.

`routes/empty_subject_security_test.go` (2026-09-29, twenty-third sweep)
pins that an empty subject is nobody: `POST /grants` with `"subject": ""` or
no subject answers 400 and stores nothing, and `GET /check` with an empty or
absent subject answers `false` even while a named subject holds a `*` grant
of the action. A mount that hands an unauthenticated caller the empty subject
therefore gets no permission from a grant nobody could have meant. Mutation-
checked: dropping the empty-subject guards from `CreateGrant` and
`MayPerform` together turns every assertion red.

`routes/retired_role_edit_security_test.go` (2026-09-29, twenty-fourth sweep)
pins that editing a retired role revives nothing: after `DELETE /roles/writer`,
`PUT /roles/writer/actions` and `PATCH /roles/writer` (body `"deleted": false`)
both answer 200, and a fresh `*` grant of it is stored, yet the role still
reads `deleted: true`, `GET /check` answers `false` for every action in every
scope and `actions_for` is empty. Mutation-checked: dropping the retired-role
predicate from `MayPerform`'s and `ActionsFor`'s joins turns four assertions red.

`routes/set_actions_wildcard_security_test.go` (2026-09-29, twenty-fifth sweep)
pins the replace door the same way `action_security_test.go` pins the add
door: `PUT /roles/writer/actions` with a wildcard (`*`, `catalog.*`), a
miscased or an empty action anywhere in the list answers 400, the role keeps
the one action it held, and a subject granted the role still may not perform
an action the refused list named. Mutation-checked: dropping `SetActions`'
`isAction` refusal turns all four refusals red.

`routes/forged_clock_security_test.go` (2026-09-30, thirty-fourth sweep) pins
that the server owns every clock: `POST /roles` claiming a 1999
`created_at`/`updated_at` and `"deleted": true`, `PATCH /roles/writer`
claiming either timestamp, `POST /roles/writer/actions` claiming a
`created_at` and `POST /grants` claiming a `granted_at` and `"deleted": true`
all store the server's own time and a live row, and the grant still permits.
`routes/everywhere_question_security_test.go` pins that `*` is a scope a
grant can hold, never one a question can ask in: a subject holding a role
only in `instance:a` answers `false` on `GET /check?scope=*` and `[]` on
`GET /subjects/{subject}/actions?scope=*`. Mutation-checked: letting
`MayPerform`'s scope predicate also match when the asked scope is `*` turns
it red. The same run measured the check under load — one subject holding
3,000 grants of a 2,000-action role answers `/check` in ~0.2 ms and
`actions_for` (18,892 bytes) in ~3 ms: one query each, not one per grant.

`grant_key_security_test.go` (2026-09-30, thirty-sixth sweep) pins that the
one-grant-per-(subject, scope, role) rule lives in the store, not only in
`CreateGrant`'s read-then-insert: a second live row for the same key,
inserted straight into the table past the pre-check (what a racing grant
would do), is refused, the table holds one row, and one `WithdrawGrant`
leaves the subject unable to act. This is what keeps revocation whole on
Postgres, where two concurrent `POST /grants` both pass the pre-check.
Mutation-checked: widening the blueprint's grant key with `granted_at`
stores the second row and turns two assertions red. (Dropping `scope`
alone from the key leaves it green — `(subject, role_code)` still
collides — and dropping every key field is refused at load.)

`routes/retired_code_reuse_security_test.go` (2026-09-30, thirty-eighth
sweep) pins that a retired role's code stays taken: after `DELETE
/roles/writer`, `POST /grants` of `writer` to `mallory` in `*` (PM10's door)
is stored, but `POST /roles {"code":"writer"}` answers `409 that role code is
taken`, and after `writer` is given `catalog.entry.delete`, `GET /check` for
`mallory` still answers `false`. A grant stored against a retired code
therefore never wakes up under a new role of the same name — which a fix for
PM10 that teaches `findRole` to skip retired rows must not undo, since
`CreateRole` uses the same lookup for its duplicate check. Mutation-checked:
skipping `CreateRole`'s duplicate check turns it red.

`routes/role_code_shape_security_test.go` (2026-10-01, fifty-first sweep)
pins that no second role can be declared that only reads like an existing
one: beside `writer`, the codes `Writer`, `writer `, ` writer`, a Cyrillic
`wrіter`, `a/b`, `..` and `writer%00` each answer `400 a role needs a code
and a name`, a grant naming `Writer`, `WRITER` or `writer ` answers `404 no
such role`, and the check for the subject stays `false`. A reviewer reading
a grant's role code therefore reads the one role it names. Mutation-checked:
reducing `isCode` to non-empty turns seven assertions red.

## Red by design

- `routes/role_seeded_security_test.go`'s
  `TestPM6_OpenHole_CreateRoleTakesSeededFromTheRequestBody` (PM6, P2): a
  role declared with `"seeded": true` in the body is stored seeded, and
  `AddAction` on it then answers 409. Leave it red until `CreateRole` stops
  taking `seeded` from the request.
- `permissions_test.go`'s
  `TestPM7_OpenBug_NameFilterTreatsUnderscoreAndPercentAsWildcards` (PM7,
  bug): `ListRoles`' `name` filter builds its `LIKE` pattern from the
  caller's search text with no escaping, so a literal `_`/`%` in the search
  over- or under-matches. Leave it red until `ListRoles` escapes both
  characters the way `mwanachama-backend-agency`'s `likeFilter` does.
- `list_cap_test.go`'s `TestPM8_OpenBug_ListRolesHasNoDefaultCap` and
  `TestPM8_OpenBug_ListGrantsHasNoDefaultCap` (PM8, bug): neither `ListRoles`
  nor `ListGrants` calls `.Limit`, so a caller who names no limit gets the
  whole live table. Leave both red until each gets a default cap.
- `action_setactions_test.go`'s
  `TestPM9_OpenBug_SetActionsRejectsADuplicateActionInTheList` (PM9, bug):
  `SetActions` inserts the caller's `actions` list verbatim with no
  de-duplication, so a repeated entry hits `role_action`'s composite primary
  key and the whole replace rolls back with a raw constraint error. Leave it
  red until `SetActions` de-duplicates before inserting.
- `grant_retired_role_test.go`'s `TestPM10_OpenBug_CreateGrantAcceptsARetiredRole`
  (PM10, bug): `CreateGrant`'s existence check (`findRole`) does not filter
  `Deleted`, so a brand-new grant can be minted against an already-retired
  role code and reads back as an ordinary live grant that `MayPerform`
  silently denies forever. Leave it red until `CreateGrant` refuses a
  retired role the way it already refuses an unknown one.
- `grant_race_test.go`'s `TestPM11_OpenBug_ConcurrentCreateGrantOnANewKeyFailsInstead`
  (PM11, bug), `role_race_test.go`'s
  `TestPM12_OpenBug_ConcurrentCreateRoleOnANewCodeFailsInstead` (PM12, bug)
  and `action_race_test.go`'s
  `TestPM13_OpenBug_ConcurrentAddActionOnTheSamePairFailsInstead` (PM13,
  bug): `CreateGrant`, `CreateRole` and `AddAction` each check-then-insert
  with no `isUniqueViolation` recovery anywhere in this repo, so two callers
  racing the same brand-new key (or, for `AddAction`, the same already-held
  one) leave one of them with a raw SQL constraint error instead of a clean
  success, a domain-level duplicate answer, or the no-op `AddAction` already
  guarantees for a non-racing repeat. Leave all three red until each insert
  recognises and recovers from its own unique violation.
- `role_order_test.go`'s
  `TestPM14_OpenBug_ListRolesIgnoresTheDeclaredNewestFirstOrder` (PM14,
  bug): `permissions.operations.json`'s `list_roles` says "newest naming
  first", but `ListRoles` orders by `Code` — caller-supplied, not
  server-minted, so it has no relationship to creation time. Leave it red
  until either `ListRoles` orders by `CreatedAt` or the operation's own
  description is corrected to describe what it actually does.

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
