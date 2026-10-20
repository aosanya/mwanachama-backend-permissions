# Permissions — why it is shaped this way

The cross-repo design record is
[architecture-permissions.md](../../../mwanachama-developer/documentation/2.%20design/architecture-permissions.md).
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
module exists to prevent. `SetActions` is what makes that affordable: a
seeding pass replaces a role's whole list from the mounted modules' declared
actions, so the list is recomputed rather than maintained. A role written
that way is marked `seeded`, and `AddAction`/`RemoveAction` then refuse it —
a hand edit would be silently undone by the next pass, which is worse than
being told no.

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

## What it does not do

It gates **actions**, not rows. "May read entries, but only pending ones" is
not expressible here and is deliberately out of scope — it belongs to the
module that owns the rows, the way catalog's `visibility` already works.

It does not decide who may grant. Granting is itself an action
(`permissions.grant.create`), so the module gates itself, and what seeds the
very first grant is a bootstrap problem for whoever mounts it.
