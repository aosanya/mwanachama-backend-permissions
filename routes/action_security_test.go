package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestAddAction_AWildcardOrMiscasedActionIsRefusedThroughTheRoute(t *testing.T) {
	mux := mountPermissions(t)
	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: a well-formed action is accepted, got %d %s", rec.Code, rec.Body.String())
	}
	for _, action := range []string{"catalog.*", "*", "Catalog.Entry.Delete", ""} {
		rec := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"`+action+`"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("adding action %q answered %d %s, want 400", action, rec.Code, rec.Body.String())
		}
	}
	listed := send(t, mux, "GET", "/permissions/roles/writer/actions", "")
	if strings.Count(listed.Body.String(), `"action"`) != 1 {
		t.Errorf("the role holds more than its one well-formed action: %s", listed.Body.String())
	}
}

func TestCreateGrant_APaddedSubjectPermitsNothingToTheTrimmedOne(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`)
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":" bob ","scope":"module:catalog","role_code":"writer"}`); rec.Code != http.StatusCreated && rec.Code != http.StatusBadRequest {
		t.Fatalf("precondition: a padded subject is either stored or refused, got %d %s", rec.Code, rec.Body.String())
	}
	rec := send(t, mux, "GET", "/permissions/check?subject=bob&scope=module:catalog&action=catalog.entry.upsert", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "false" {
		t.Errorf("a grant made to %q answered the check for %q with %d %s, want 200 false", " bob ", "bob", rec.Code, rec.Body.String())
	}
}
