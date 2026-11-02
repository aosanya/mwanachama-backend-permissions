# Permissions — why it is shaped this way

The cross-repo design record is
[architecture-permissions.md](../../../developer/documentation/2.%20design/architecture-permissions.md).
This page carries what would otherwise be comments in the source.

## It names no domain, and that is the point

Nothing in this module knows what an action means. `permissions.grant.create`
and `x_ray.order.approve` are both opaque strings compared for equality. The
same store holds a studio's operator roles and a hospital's clinical
privileges, which is why `spec/examples/` carries both and
`domain_agnostic_test.go` fails the build on a word that means something in
only one of them.

The rule is one sentence:

> A word that means something in one domain and nothing in another does not
> belong in this module.

`role`, `grant`, `action` and `scope` pass it. `operator`, `tenant` and
`ward` do not, and the test lists them.

## The three fields that carry meaning

**`subject`** is whoever authenticated, verbatim. This module never resolves
it to a person and never validates its shape beyond non-empty — the same
posture `agency_instance_members.UserID` already takes.

**`scope`** is an opaque key naming where a grant applies, validated against
`^\*$|^[a-z][a-z0-9_]*:[A-Za-z0-9_.:/-]+$`. One value is read rather than
compared: `*` applies everywhere. The pattern exists because a scope reaches
a comparison as data, and an unvalidated one is a row nobody can audit.

**`partition`** (PM19) names which partition of the deployment a grant holds
in — for a host that runs many independent tenants over one permissions
store, the tenant the grant was made in. Like `scope` it is an opaque key the
caller chooses, validated against `^[A-Za-z0-9][A-Za-z0-9_.:-]*$`, and the
module never reads it for meaning. It is part of the grant's key, so one
subject may hold the same role on the same scope in two partitions and lose
one without the other.

**Empty means every partition.** That is what every grant made before PM19
means, and it keeps meaning it: an unpartitioned grant answers a question
asked in any partition and one asked in none. A grant naming a partition
answers only a question asked in that partition — never one asked in another,
and never one asked outside every partition. So:

| The grant names | Asked in `alpha` | Asked in `beta` | Asked in no partition |
|---|---|---|---|
| no partition | yes | yes | yes |
| `alpha` | yes | no | no |

`MayPerformIn(subject, partition, scope, action)` is the gate's question with
a partition; `MayPerform(subject, scope, action)` is the same question asked
in no partition, so every existing caller keeps its exact behaviour.
`ActionsFor` lists in no partition too, matching `MayPerform`.
`ActionsAnywhere` counts every grant, partitioned or not, because it answers
what a subject might be offered somewhere, not what the gate will let through
here. `WithdrawGrantIn` takes a partitioned grant back; `WithdrawGrant` takes
back the one naming none. Over HTTP both the check and the withdrawal take an
optional `partition` query parameter, and a grant carries `partition` in its
body.

**An existing grant table gains the column on the next `Migrate`.**
`spec.Migrate` only creates what is absent (shared's S52), so `Migrate` adds
`partition text not null default ''` to a grant table that lacks it — every
old row reads back as naming no partition — and on Postgres re-keys the table
to `(subject, scope, partition, role_code)`. SQLite cannot change a primary
key in place; a SQLite store from before PM19 keeps its old key, which only
matters once one subject is granted the same role on the same scope in two
partitions.

**`action`** is a declared action, validated against
`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`. `MayPerform` refuses a malformed
one rather than returning false, because a caller asking about `NOPE` has a
bug and a silent `false` hides it.

## Decisions

**Additive only — there are no denies.** A subject may perform an action if
some grant permits it, and nothing subtracts. Deny rules need a precedence
order, and precedence is where access-control systems stop being possible to
reason about. To take something away, withdraw the grant or move the subject
to a smaller role.

**Actions are listed one by one, never matched by pattern.** No
`catalog.*.list`. A pattern is one row instead of twelve and it widens itself
the moment a module declares a matching operation — which is the failure this
module exists to prevent. `SeedActions` is what makes that affordable: a
seeding pass replaces a role's whole list from the mounted modules' declared
actions, so the list is recomputed rather than maintained. A role written
that way is marked `seeded`, and `AddAction`/`RemoveAction` then refuse it —
a hand edit would be silently undone by the next pass, which is worse than
being told no.
`seeded` is server-owned, and only the host's own seeding pass sets it
(PM6, decision 444). `CreateRole` always writes it false whatever the
request body says, so a hand-made role cannot arrive already locked.
`SeedActions` (and `Seed`, which calls it) is a Go method only: it is not a
declared operation, so no route and no MCP tool reaches it. The declared
`set_actions` operation (`PUT /roles/{role_code}/actions`) calls
`SetActions`, which replaces the list the same way but never marks the role
seeded — otherwise any caller allowed to replace a list could lock a
hand-made role against `AddAction`/`RemoveAction`. It does not refuse an
already-seeded role either; replacing that list leaves the mark as it was.

**Retiring a role stops every grant naming it, and the grants stay.**
`MayPerform` joins through the role and checks `deleted` on both sides. The
grants are kept because a withdrawn permission and a forgotten one look
identical afterwards, and only one of them can be explained.

**A withdrawn grant is reinstated, not duplicated.** Subject, scope and role
are the whole primary key, so `CreateGrant` on a withdrawn row turns it back
on. A fuller audit trail belongs wherever the act is recorded, not in the
table the gate reads on every call.

**`MayPerform` is the gate; `ActionsFor` is a convenience.** The second
answers "which tools should this caller be offered", which is a listing
decision. Enforcement is always the refusal on call. Swapping them would make
a filtered menu the security boundary, which it is not.

## Acting for somebody else is its own action

A grant to do something for other people is a separate action id, held by a
separate role, never a widening of the action that does it for oneself. The
member app's field worker (member app decision 240, PM20) is the first case:
answering one's own sheet and collecting sheets from people without a phone
are two declared actions, and only the role that lists the second lets its
holder submit for anyone else. A grant to answer therefore never carries a
grant to collect, and withdrawing the collecting grant leaves the holder's
own answering alone if another role still lists it.

**The words live with the host, not here.** This module stores the pair and
compares strings, so nothing in it names the action or the role. The action
is declared in the forms module's operations file, which owns the operation
it gates (forms F36), and the role that holds it, with its name and its
seeding, is the host's — wakala-api's (WK169), alongside the other roles it
seeds. `collect_grant_test.go` holds the shape with neutral names: a
collecting role reaches the collect action in its granted scope only, an
answering role never reaches it, and withdrawal stops it.

## A grant scoped to one record, with an end (PM22)

The member app's outsiders (decisions 296 and 299 there: a client, a guardian,
a bidder) see only what is linked to them, and lose it when the record closes
or staff end it. Two additions carry that, and neither widens anything.

**A record scope.** A scope of the form `record:<module>/<object>/<id>` names
one record. The module and object segments are lower-case codes, the id is
`[A-Za-z0-9_.:-]+`, and any scope beginning `record:` that is not exactly that
shape is refused as `ErrInvalidScope` — no empty segment, no fourth segment, no
`*`. It is compared exactly like every other scope, so a record grant answers a
check on that record and nothing else: not a sibling record, not
`module:<module>`, not a prefix of its own path. Nothing here walks from a
record up to its module or object; a host that wants both asks twice. The one
reserved scope `*` still answers everywhere, records included, as it always
has.

**An end.** `ends_at` is optional RFC 3339 text. `CreateGrant` parses it,
refuses an unreadable one or one not in the future as `ErrInvalidEnd` (400),
and stores it in UTC at a fixed nanosecond width so the database compares it as
text in time order. Empty means the grant never ends. Every read that honours a
grant — `MayPerform`, `MayPerformIn`, `ActionsFor`, `ActionsAnywhere` and
`ListGrants` — keeps only `ends_at = '' or ends_at > now`, so past its end a
grant stops answering and stops listing, exactly as if it had been withdrawn.
The row itself is kept, like a withdrawn one, so the trail survives.
Reinstating a withdrawn grant takes the end the new request carries.

Withdrawal is unchanged: `WithdrawGrant` with the record scope ends it at once.
`Migrate` adds the `ends_at` column to a grant table that predates it, the same
way it adds `partition`.

Still not here: no wildcard (a record grant never matches by pattern), no deny
rule (an end subtracts nothing, the grant simply stops adding), and no
notification — telling the outsider their access ended (decision 299) is the
host's.

## Reading one person's access in one partition (HS653)

A page of what one person can do needs the role behind every action and only
the agency being looked at. No earlier read gave that. `ActionsFor` and
`ActionsAnywhere` return bare action strings with no role; `ActionsFor` sees
only the unpartitioned grants; `ActionsAnywhere` takes no partition; and
`ListGrants` and `ListActions` stop at 100 rows with no cursor.

`AccessOf(subject, partition)` is the one read for it, declared as
`subject_access` (`GET /subjects/{subject}/access?partition=`) and gated by its
own action, `permissions.subject.access`. It runs the same join of grants,
roles and role actions as `MayPerformIn`, so what it lists is what the gate
would allow in that partition.

**The partition is required.** A missing, empty or malformed one is
`ErrInvalidPartition`, a 400 over HTTP. Asking in no partition would list only
the unpartitioned grants and hide which agency the rest belong to.

**Which grants count.** Those naming the asked partition and those naming none,
which apply everywhere. A grant naming another partition is never read. A
record-scoped grant is listed with its `record:` scope.

**The line.** One line per grant and action, so an action given by two roles
appears twice, once for each role, and a page can name the role that gives each
one. A line carries `action`, `role_code`, `role_name`, `scope`, `partition`
(empty for an unpartitioned grant), `granted_by`, `granted_at` and `ends_at`
(empty when the grant never ends). Lines come ordered by action, then role,
scope and partition. A subject holding nothing there gets `[]`, never `null`.

**What is left out.** A retired role, a withdrawn grant and a grant past its
`ends_at`, exactly as the gate leaves them out. Nothing is cut at a page size:
the read returns every line.

**Why `list_grants` with only a subject stays operator-only.** Its partition
filter is an exact match, and when it is absent the read spans every
partition, so a subject filter alone returns that person's grants in all of
them. A manager of agency A who held `permissions.grant.list` would read
agency B's grants too. The filter is left as it is, because an operator
listing the whole store is what the read is for, so `permissions.grant.list`
belongs to operators and is not to be given to an agency's manager. A per-agency page calls `subject_access` with its
agency instead, which cannot cross a partition.

## What it does not do

It gates **actions**, not rows. "May read entries, but only pending ones" is
not expressible here and is deliberately out of scope — it belongs to the
module that owns the rows, the way catalog's `visibility` already works.

It does not decide who may grant. Granting is itself an action
(`permissions.grant.create`), so the module gates itself, and what seeds the
very first grant is a bootstrap problem for whoever mounts it.

## Permissions per role kind, in two layers (PM21)

A **permission** is a code a host names, such as the ones agency's AG88 catalog
lists (`run_poll`, `post_notice`, `post_event`, `post_volunteer_call`); the
actions a permission allows are that catalog's to say. This module stores only
whether a **role kind** holds a permission in one **partition** (an agency), and
which layer said so. It never learns the permission's name or its actions.

The `permission` object is keyed by `(partition, permission, role_kind, layer)`, and
`layer` is `definition` or `override`:

- `SetPermissionDefinition` writes the published definition's tick. A publish
  rewrites it and never touches an override, and `set_by` is always empty.
- `OverridePermission` writes the owner's tick, with `set_by` required. It is the
  caller's to fill from the session, never from a request body.
- `ReadPermission` answers the override when one exists, else the definition, and
  the answer carries `layer`, `set_by` and `set_at`, so a screen can say who
  changed a permission and when. Neither layer is `ErrPermissionNotFound`.
- `ClearPermissionOverride` removes the override, and the definition answers again.
- `ListPermissions` answers every `(permission, role_kind)` pair with a row in one
  partition, ordered by permission then role kind, each read exactly as
  `ReadPermission` would: the override when one exists, else the definition, with
  its `layer`, `set_by` and `set_at`. It is the studio's tick screen in one
  read. A pair with no row in either layer is not listed, and an empty
  partition answers an empty list.

This object was called `power` until 2026-10-31. A deployment that stored it
under the old names keeps its rows: `Migrate` renames the `powers` table to
`permissions` and its `power` column to `permission` before anything else, and
runs again without effect. The word is shared with the `permissions.*` action
names on purpose, but the two are different things: an action is what an
operation carries, and a permission here is a host's code that names a list of
actions elsewhere, ticked per role kind. `TestMigrateCarriesAPowersTableAndItsPowerColumnOverToPermissions`
pins the rename.

`held` is a value per layer, not a deny rule: no grant subtracts from
another, and an override replaces the definition's answer for exactly one
key. The `permission` role is optional in a domain spec; a spec that fills none
gets `ErrPermissionsNotDeclared` from these methods, so an existing deployment's
spec keeps loading unchanged.

Four of the five have addresses: `GET`, `PUT` and `DELETE` on
`/permissions/{partition}/{permission}/{role_kind}`, gated by `permissions.permission.read`,
`permissions.permission.override` and `permissions.permission.clear`, and `GET
/permissions/{partition}` for `ListPermissions`, gated by its own
`permissions.permission.list`; the `PUT` binds `set_by` `from: caller`.

An override may untick a permission the definition ticks: `held` is stored per
layer and the override's value answers, true or false. The owner settled it
(member app decision 268, 2026-10-22: owners may override any unlocked tick),
and `TestAnOverrideWinsOverTheDefinitionAndSaysWhoSetIt` pins it. Which ticks
are locked is the host's to say and to enforce (wakala-api WK179); this module
stores no lock. `SetPermissionDefinition` has no address on purpose: a
publish writes it in process.

**A held permission is not folded into the gate; that is the host's.** The
actions a permission allows are AG88's catalog to say, and this module never
learns them, so `MayPerformIn`, `MayPerform`, `ActionsFor` and
`ActionsAnywhere` read grants and role actions only. A permission row grants no
action, even when its role kind is a role the subject is granted; `TestAHeldPermissionGrantsNoActionThroughTheGate`
pins it. A host that gates a permission's actions asks `ReadPermission` for the
caller's role kinds and maps the permission to its actions itself. Who answers
when no role kind holds a permission (owners and agents, decision 342/348) is the
host's rule over `ReadPermission`, not stored here.
