package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestSetActions_AWildcardAnywhereInTheListIsRefusedAndTheOldListStands(t *testing.T) {
	mux := mountPermissions(t)
	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "PUT", "/permissions/roles/writer/actions", `{"actions":["catalog.entry.upsert"]}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: a well-formed replace answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"bob","scope":"module:catalog","role_code":"writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: grant answered %d %s", rec.Code, rec.Body.String())
	}

	for _, list := range []string{
		`["catalog.entry.delete","*"]`,
		`["catalog.*"]`,
		`["catalog.entry.upsert","Catalog.Entry.Delete"]`,
		`["catalog.entry.upsert",""]`,
	} {
		if rec := send(t, mux, "PUT", "/permissions/roles/writer/actions", `{"actions":`+list+`}`); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT actions %s answered %d %s, want 400", list, rec.Code, rec.Body.String())
		}
	}

	listed := send(t, mux, "GET", "/permissions/roles/writer/actions", "")
	if strings.Count(listed.Body.String(), `"action"`) != 1 || !strings.Contains(listed.Body.String(), `"catalog.entry.upsert"`) {
		t.Errorf("a refused replace changed the role's actions: %s", listed.Body.String())
	}
	check := send(t, mux, "GET", "/permissions/check?subject=bob&scope=module:catalog&action=catalog.entry.delete", "")
	if check.Code != http.StatusOK || strings.TrimSpace(check.Body.String()) != "false" {
		t.Errorf("after refused wildcard replaces, bob may catalog.entry.delete: %d %s, want 200 false", check.Code, check.Body.String())
	}
}
