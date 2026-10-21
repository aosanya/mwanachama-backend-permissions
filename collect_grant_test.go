package mwanachamapermissions_test

import (
	"context"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

const (
	answerOwn      = "intake.answer.submit"
	answerOnBehalf = "intake.answer.collect"
	collectorRole  = "collector"
	answererRole   = "answerer"
)

func seedCollectingRoles(t *testing.T, mgr permissions.PermissionsManager, ctx context.Context) {
	t.Helper()
	for code, actions := range map[string][]string{
		answererRole:  {answerOwn},
		collectorRole: {answerOwn, answerOnBehalf},
	} {
		if _, err := mgr.CreateRole(ctx, permissions.Role{Code: code, Name: code}); err != nil {
			t.Fatalf("CreateRole %s: %v", code, err)
		}
		if _, err := mgr.SetActions(ctx, code, actions); err != nil {
			t.Fatalf("SetActions %s: %v", code, err)
		}
	}
}

func mayPerform(t *testing.T, mgr permissions.PermissionsManager, ctx context.Context, subject, scope, action string) bool {
	t.Helper()
	ok, err := mgr.MayPerform(ctx, subject, scope, action)
	if err != nil {
		t.Fatalf("MayPerform %s %s %s: %v", subject, scope, action, err)
	}
	return ok
}

func TestCollectingForOthers_IsItsOwnActionHeldOnlyWithTheCollectingRole(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedCollectingRoles(t, mgr, ctx)

	for subject, role := range map[string]string{alice: collectorRole, bob: answererRole} {
		if _, err := mgr.CreateGrant(ctx, permissions.Grant{Subject: subject, Scope: shop, RoleCode: role}); err != nil {
			t.Fatalf("CreateGrant %s: %v", role, err)
		}
	}

	if !mayPerform(t, mgr, ctx, alice, shop, answerOnBehalf) {
		t.Fatal("the collecting role's holder may not collect for others")
	}
	if !mayPerform(t, mgr, ctx, alice, shop, answerOwn) {
		t.Fatal("the collecting role's holder lost their own answering")
	}
	if mayPerform(t, mgr, ctx, bob, shop, answerOnBehalf) {
		t.Fatal("a grant to answer carried a grant to collect for others")
	}
	if !mayPerform(t, mgr, ctx, bob, shop, answerOwn) {
		t.Fatal("the plain answerer may not answer")
	}
	if mayPerform(t, mgr, ctx, alice, other, answerOnBehalf) {
		t.Fatal("collecting reached a scope the grant does not name")
	}
}

func TestCollectingForOthers_StopsWhenTheGrantIsWithdrawn(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedCollectingRoles(t, mgr, ctx)

	if _, err := mgr.CreateGrant(ctx, permissions.Grant{Subject: alice, Scope: shop, RoleCode: collectorRole}); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if err := mgr.WithdrawGrant(ctx, alice, shop, collectorRole); err != nil {
		t.Fatalf("WithdrawGrant: %v", err)
	}
	if mayPerform(t, mgr, ctx, alice, shop, answerOnBehalf) {
		t.Fatal("a withdrawn collecting grant still lets its holder collect")
	}
}
