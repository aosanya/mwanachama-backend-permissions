# Design

[permissions.md](permissions.md) is the whole of this module's design, and
the page that carries what would otherwise be comments in the source:

- **It names no domain** — an action is an opaque string compared for
  equality, enforced by `domain_agnostic_test.go`.
- **The three fields that carry meaning** — `subject` verbatim, `scope` an
  opaque key, `action` a declared operation id.
- **Decisions** — no deny rules, actions listed one by one and never by
  pattern, a retired role stopping every grant that names it, a withdrawn
  grant reinstated rather than duplicated, and `MayPerform` as the gate where
  `ActionsFor` is only a listing.
- **What it does not do** — gate rows rather than actions, or seed the first
  grant; that bootstrap is the mount's.

The cross-repo record it implements is
[architecture-permissions.md](../../../developer/documentation/2.%20design/architecture-permissions.md).
The objects are declared in `permissions.blueprint.json` and the routes in
`permissions.operations.json`, both at the repo root; the engine that reads
them is `mwanachama-backend-shared`'s `spec` and `dispatch`, documented in
that repo's own `2. design/`.

The security sweep checks these decisions against the shipped code every
run; what it has pinned is listed in [`4. qa/README.md`](../4.%20qa/README.md).
