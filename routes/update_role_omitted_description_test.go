package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestPM17_OpenBug_UpdateRoleClobbersDescriptionWhenTheBodyOmitsIt(t *testing.T) {
	mux := mountPermissions(t)
	create := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer","description":"can write entries"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", create.Code, create.Body.String())
	}

	rename := send(t, mux, "PATCH", "/permissions/roles/writer", `{"name":"Senior Writer"}`)
	if rename.Code != http.StatusOK {
		t.Fatalf("PATCH /permissions/roles/writer (name only) answered %d %s, want 200", rename.Code, rename.Body.String())
	}

	got := send(t, mux, "GET", "/permissions/roles/writer", "")
	var role map[string]any
	if err := json.Unmarshal(got.Body.Bytes(), &role); err != nil {
		t.Fatalf("GET /permissions/roles/writer body %q: %v", got.Body.String(), err)
	}

	if role["description"] != "can write entries" {
		t.Fatalf("PM17 open bug: PATCH /permissions/roles/writer with body %s (no \"description\" key) wiped the role's description to %q; want it to stay %q, because the operation's own declared description says the caller may \"change a bundle's name or its description\" independently, and this PATCH only asked to rename it", `{"name":"Senior Writer"}`, role["description"], "can write entries")
	}
}
