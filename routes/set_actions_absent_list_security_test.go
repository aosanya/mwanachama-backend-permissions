package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestPM14_OpenHole_ReplaceWithNoActionsKeyStripsEveryActionFromTheRole(t *testing.T) {
	mux := mountPermissions(t)
	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"owner","name":"Owner"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "PUT", "/permissions/roles/owner/actions", `{"actions":["permissions.grant.create"]}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: replace actions answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"alice","scope":"*","role_code":"owner"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: grant answered %d %s", rec.Code, rec.Body.String())
	}
	const check = "/permissions/check?subject=alice&scope=*&action=permissions.grant.create"
	if rec := send(t, mux, "GET", check, ""); strings.TrimSpace(rec.Body.String()) != "true" {
		t.Fatalf("precondition: alice's check answered %d %s, want true", rec.Code, rec.Body.String())
	}

	for _, body := range []string{`{"action":["permissions.grant.create"]}`, `{}`, `{"actions":null}`} {
		rec := send(t, mux, "PUT", "/permissions/roles/owner/actions", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("PM14 (P2): PUT /roles/owner/actions %s answered %d %s; a replace that names no actions list must be refused 400, as the declared required argument says, not read as clear the role", body, rec.Code, strings.TrimSpace(rec.Body.String()))
		}
	}

	if rec := send(t, mux, "GET", check, ""); strings.TrimSpace(rec.Body.String()) != "true" {
		t.Errorf("PM14 (P2): after replaces that named no actions list alice's check answers %s; want true, the role must keep the actions nobody listed", strings.TrimSpace(rec.Body.String()))
	}
}
