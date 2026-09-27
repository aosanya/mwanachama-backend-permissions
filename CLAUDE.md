# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repository.

## Project: mwanachama-backend-permissions

Who may perform which declared action, where. A Go library backed by
Postgres or SQLite through GORM, with its objects declared rather than
written. There is no service to run — `Migrate(db, spec)` creates the tables
and `NewPermissionsManager(db, spec)` returns the manager.

Built 2026-09-27 on [`mwanachama-backend-catalog`](../mwanachama-backend-catalog)'s
shape, at the owner's instruction. Read that repo before changing this one:
blueprint plus domain spec, `specstore` rather than row structs, operations
declared in JSON and dispatched into routes and MCP tools, a
`domain_agnostic_test.go` that fails the build on a domain word.

**It was not a conversion of `mwanachama-backend-auth`.** That was the first
plan and was ruled out: Supabase holds authentication, and auth's storage
needs five things `mwanachama-backend-shared/spec` does not express — a bytes
type, SQL NULL, a partial unique index on `retired_at IS NULL`, Postgres
SEQUENCEs for its ids, and `time.Time` columns where the engine stores
timestamps as text. Permissions needs none of them.

## It names no domain, and that is the point

Nothing here knows what an action means. `permissions.grant.create` and
`x_ray.order.approve` are both opaque strings compared for equality.

> A word that means something in one domain and nothing in another does not
> belong in this module.

`domain_agnostic_test.go` enforces it by parsing this package's identifiers
and its blueprint, and `spec/examples/` carries two domains that disagree
about every noun so the claim is tested rather than asserted.

## Conventions

- **The objects are declared, not written.** `permissions.blueprint.json`
  declares every object, field and index; a domain spec names them; the
  engine is `mwanachama-backend-shared/spec` and `/specstore`. There are no
  row structs and no `AutoMigrate`. Adding a field means editing the
  blueprint and the `models/` type together — they are checked against each
  other when the manager is built.
- **A table is `<instance>_permissions_<table>`.**
- **Every operation carries an `action`**, because that is what this module
  stores. `AnonymousActions` is deliberately empty: nothing here is readable
  without presenting a caller.
- **No wildcards in `role_action`.** A pattern widens itself the next time
  somebody declares an operation. `SetActions` is what keeps enumeration
  affordable — see [documentation/2. design/permissions.md](documentation/2.%20design/permissions.md).
- **There are no deny rules.** Grants only add. A rule that subtracts needs a
  precedence order, and precedence is where this stops being reasonable
  about.
- Documentation follows the workspace-wide four-phase layout
  (`1. requirements`/`2. design`/`3. implementation`/`4. qa`).

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
