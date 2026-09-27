# mwanachama-backend-permissions (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

The cross-repo rows that put this module into service — the `dispatch` gate,
the seeding pass, the wakala-api mount, the membership migration, the
`tools/list` filter and the Modules screen — are `DSN-1734`…`DSN-1739` on
`developer`'s design board; see
[architecture-permissions.md](../../../developer/documentation/2.%20design/architecture-permissions.md).

| Task | Title | Status | Notes |
|------|-------|--------|-------|
| PM1 | 🔌 **MCP tools, the other half of catalog's shape.** | 📋 | `routes/tools.go` does not exist yet. Catalog's equivalent is what turns the declared operations into MCP tools with per-field prose, required-ness and read-only flags read from the blueprint (`routes.Fields()`, keyed by Go type), so a tool's schema and the column a write lands in cannot drift. The 14 operations and their errors are already declared, so this is `dispatch.Tools(s, deps)` plus the `Fields()` map and an `operatorOnly` equivalent — except that here nothing is anonymous, so the split is simpler than catalog's. |
| PM2 | 🧪 **No Postgres integration test.** | 📋 | Everything runs on SQLite today. Catalog carries `postgres_integration_test.go`, overriding `Instance` to a random `pg<hex>` per run so it cannot touch a real deployment — copy that discipline, not just the file. The case that matters here is `MayPerform`'s three-way join under a real planner, and the composite primary keys, which SQLite is laxer about. |
| PM3 | 🃏 **`ActionsFor` has no scope wildcard story for listing.** | 📋 | `MayPerform` reads `*` correctly. `ActionsFor` does too, but a caller asking "what may this subject do *anywhere*" has no way to ask — passing `*` as the scope means "the grants made everywhere", not "the union across every scope". Whether the second question is worth answering depends on whether a screen ever needs it; decide before adding a parameter nobody wanted. |
| PM4 | 🔒 **A grant's scope is validated; a role's actions are not checked against anything real.** | 📋 | `isAction` proves an action is well-formed, never that it exists. A typo (`agency.module.lst`) is stored happily and silently permits nothing, which is the failure mode hardest to notice. The seeding pass (`DSN-1735`) makes this mostly moot for seeded roles, but a hand-made role can still hold a permission for an action nobody declares. Options: refuse unknown actions at write time, which needs the module to be told the declared set; or a report that lists permissions matching no declared action, which keeps this module ignorant and is probably the right shape. |

_Completed rows move to [todo_done.md](todo_done.md)._
