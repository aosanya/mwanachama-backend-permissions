package routes_test

import (
	"net/http"
	"testing"
)

func TestRemovedActionStopsAuthorizingThroughAnExistingGrant(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"editor","name":"Editor"}`)
	send(t, mux, "POST", "/permissions/roles/editor/actions", `{"action":"catalog.entry.upsert"}`)
	send(t, mux, "POST", "/permissions/roles/editor/actions", `{"action":"catalog.entry.delete"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"erin","scope":"module:catalog","role_code":"editor"}`)
	if got := mayPerform(t, mux, "erin", "module:catalog", "catalog.entry.delete"); got != "true" {
		t.Fatalf("precondition: erin's grant authorizes catalog.entry.delete, got %s", got)
	}

	if rec := send(t, mux, "DELETE", "/permissions/roles/editor/actions/catalog.entry.delete", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("precondition: remove action answered %d %s", rec.Code, rec.Body.String())
	}
	if got := mayPerform(t, mux, "erin", "module:catalog", "catalog.entry.delete"); got != "false" {
		t.Errorf("an action removed from erin's role still authorizes through her grant: check answered %s", got)
	}
	if got := mayPerform(t, mux, "erin", "module:catalog", "catalog.entry.upsert"); got != "true" {
		t.Errorf("removing one action withdrew the role's other action too: check answered %s", got)
	}
}

func TestReplacedActionListStopsAuthorizingWhatItDropped(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"curator","name":"Curator"}`)
	if rec := send(t, mux, "PUT", "/permissions/roles/curator/actions", `{"actions":["catalog.entry.upsert","catalog.entry.delete"]}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: first replace answered %d %s", rec.Code, rec.Body.String())
	}
	send(t, mux, "POST", "/permissions/grants", `{"subject":"finn","scope":"*","role_code":"curator"}`)
	if got := mayPerform(t, mux, "finn", "module:catalog", "catalog.entry.delete"); got != "true" {
		t.Fatalf("precondition: finn's everywhere grant authorizes catalog.entry.delete, got %s", got)
	}

	if rec := send(t, mux, "PUT", "/permissions/roles/curator/actions", `{"actions":["catalog.entry.upsert"]}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: narrowing replace answered %d %s", rec.Code, rec.Body.String())
	}
	if got := mayPerform(t, mux, "finn", "module:catalog", "catalog.entry.delete"); got != "false" {
		t.Errorf("an action a replace dropped still authorizes through finn's grant: check answered %s", got)
	}
}
