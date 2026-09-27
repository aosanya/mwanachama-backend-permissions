package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestSetActions_AWildcardMiscasedOrPaddedActionIsRefusedAndNothingIsStored(t *testing.T) {
	mux := mountPermissions(t)
	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: creating the role answered %d %s", rec.Code, rec.Body.String())
	}

	for _, body := range []string{
		`{"actions":["catalog.*"]}`,
		`{"actions":["*"]}`,
		`{"actions":["Catalog.Entry.List"]}`,
		`{"actions":["catalog.entry.list"," catalog.entry.read"]}`,
	} {
		if rec := send(t, mux, "PUT", "/permissions/roles/writer/actions", body); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT /roles/writer/actions %s answered %d %s; it should be refused 400", body, rec.Code, rec.Body.String())
		}
	}
	if rec := send(t, mux, "GET", "/permissions/roles/writer/actions", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("a refused replacement left actions behind: %s", rec.Body.String())
	}
}

func TestDeletedRole_NewActionsAndGrantsOnItAuthorizeNothingAndItStaysDeleted(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.list"}`)
	if rec := send(t, mux, "DELETE", "/permissions/roles/writer", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("precondition: deleting the role answered %d %s", rec.Code, rec.Body.String())
	}

	send(t, mux, "PATCH", "/permissions/roles/writer", `{"name":"Back","deleted":false}`)
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.read"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"bob","scope":"instance:a","role_code":"writer"}`)

	for _, action := range []string{"catalog.entry.list", "catalog.entry.read"} {
		if got := mayPerform(t, mux, "bob", "instance:a", action); got != "false" {
			t.Errorf("a grant on a deleted role permits %s: %s", action, got)
		}
	}
	if rec := send(t, mux, "GET", "/permissions/subjects/bob/actions?scope=instance:a", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("a grant on a deleted role lists actions: %s", rec.Body.String())
	}
	if rec := send(t, mux, "GET", "/permissions/roles/writer", ""); !strings.Contains(rec.Body.String(), `"deleted":true`) {
		t.Errorf("a PATCH claiming deleted:false revived the role: %s", rec.Body.String())
	}
}
