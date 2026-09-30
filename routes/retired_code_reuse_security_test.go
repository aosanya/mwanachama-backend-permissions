package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestARetiredRoleCodeCannotBeRecreatedSoAGrantStoredAgainstItStaysDormant(t *testing.T) {
	mux := mountPermissions(t)
	const check = "/permissions/check?subject=mallory&scope=instance:abc&action=catalog.entry.delete"

	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "DELETE", "/permissions/roles/writer", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("precondition: retire role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"mallory","scope":"*","role_code":"writer"}`); rec.Code != http.StatusCreated && rec.Code != http.StatusConflict && rec.Code != http.StatusNotFound {
		t.Fatalf("precondition: a grant on the retired code answered %d %s", rec.Code, rec.Body.String())
	}

	recreate := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer v2"}`)
	if recreate.Code != http.StatusConflict {
		t.Errorf("re-creating the retired role code writer answered %d %s, want 409 so no grant stored against the old code is inherited", recreate.Code, recreate.Body.String())
	}
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.delete"}`)

	if rec := send(t, mux, "GET", check, ""); strings.TrimSpace(rec.Body.String()) != "false" {
		t.Errorf("a grant stored against a retired role code permitted catalog.entry.delete after the code was re-created and given that action: %d %s, want false", rec.Code, rec.Body.String())
	}
}
