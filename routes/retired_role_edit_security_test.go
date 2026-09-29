package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRetiredRoleEditedThroughTheRoutesStaysRetiredAndPermitsNothing(t *testing.T) {
	mux := mountPermissions(t)
	const check = "/permissions/check?subject=alice&scope=instance:abc&action=catalog.entry.upsert"

	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: add action answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"alice","scope":"instance:abc","role_code":"writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create grant answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "GET", check, ""); strings.TrimSpace(rec.Body.String()) != "true" {
		t.Fatalf("precondition: the live grant should permit the action, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "DELETE", "/permissions/roles/writer", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("precondition: retire role answered %d %s", rec.Code, rec.Body.String())
	}

	send(t, mux, "PUT", "/permissions/roles/writer/actions", `{"actions":["catalog.entry.upsert","permissions.grant.create"]}`)
	send(t, mux, "PATCH", "/permissions/roles/writer", `{"name":"Revived","deleted":false}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"alice","scope":"*","role_code":"writer"}`)

	read := send(t, mux, "GET", "/permissions/roles/writer", "")
	var role struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &role); err != nil || read.Code != http.StatusOK {
		t.Fatalf("read role answered %d %s (%v)", read.Code, read.Body.String(), err)
	}
	if !role.Deleted {
		t.Errorf("a retired role edited through PATCH and PUT read deleted=false; editing must never revive a retired role: %s", read.Body.String())
	}
	for _, action := range []string{"catalog.entry.upsert", "permissions.grant.create"} {
		for _, scope := range []string{"instance:abc", "instance:other"} {
			rec := send(t, mux, "GET", "/permissions/check?subject=alice&scope="+scope+"&action="+action, "")
			if strings.TrimSpace(rec.Body.String()) != "false" {
				t.Errorf("check %s in %s on a retired role after PATCH, PUT and a * grant answered %d %s, want false", action, scope, rec.Code, rec.Body.String())
			}
		}
	}
	if rec := send(t, mux, "GET", "/permissions/subjects/alice/actions?scope=instance:abc", ""); strings.TrimSpace(rec.Body.String()) != "[]" && strings.TrimSpace(rec.Body.String()) != "null" {
		t.Errorf("actions_for on a retired role answered %d %s, want an empty list", rec.Code, rec.Body.String())
	}
}
