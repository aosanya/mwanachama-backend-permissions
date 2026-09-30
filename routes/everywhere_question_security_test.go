package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestAskingInTheEverywhereScopeDoesNotWidenAScopedGrant(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`)
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"member-a","scope":"instance:a","role_code":"writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create grant answered %d %s", rec.Code, rec.Body.String())
	}
	if got := mayPerform(t, mux, "member-a", "instance:a", "catalog.entry.upsert"); got != "true" {
		t.Fatalf("precondition: the grant's own scope answered %s, want true", got)
	}

	if got := mayPerform(t, mux, "member-a", "*", "catalog.entry.upsert"); got != "false" {
		t.Errorf("a check asked in scope * answered %s for a grant held only in instance:a, want false", got)
	}
	rec := send(t, mux, "GET", "/permissions/subjects/member-a/actions?scope=*", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("actions_for asked in scope * answered %d %s for a grant held only in instance:a, want 200 []", rec.Code, rec.Body.String())
	}
}
