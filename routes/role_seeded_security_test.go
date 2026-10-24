package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestPM6_CreateRoleIgnoresSeededInTheRequestBody(t *testing.T) {
	mux := mountPermissions(t)
	rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer","seeded":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s, want 201", rec.Code, rec.Body.String())
	}

	read := send(t, mux, "GET", "/permissions/roles/writer", "")
	var role struct {
		Seeded bool `json:"seeded"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &role); err != nil || read.Code != http.StatusOK {
		t.Fatalf("precondition: read role answered %d %s (%v)", read.Code, read.Body.String(), err)
	}
	if role.Seeded {
		t.Errorf("PM6 (P2) open hole: a role created by hand reads seeded=true because the request body said so; a role is seeded only when a seeding pass writes it, so it should read seeded=false")
	}

	add := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`)
	if add.Code != http.StatusCreated {
		t.Errorf("PM6 (P2) open hole: adding an action to a hand-made role answered %d %s, want 201; the body's seeded=true locked it as if a seeding pass owned it", add.Code, add.Body.String())
	}
}

func TestPM6_SetActionsThroughTheRouteTableLeavesTheRoleUnseeded(t *testing.T) {
	mux := mountPermissions(t)
	rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s, want 201", rec.Code, rec.Body.String())
	}

	set := send(t, mux, "PUT", "/permissions/roles/writer/actions", `{"actions":["catalog.entry.upsert"]}`)
	if set.Code != http.StatusOK {
		t.Fatalf("set_actions answered %d %s, want 200", set.Code, set.Body.String())
	}

	read := send(t, mux, "GET", "/permissions/roles/writer", "")
	var role struct {
		Seeded bool `json:"seeded"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &role); err != nil || read.Code != http.StatusOK {
		t.Fatalf("precondition: read role answered %d %s (%v)", read.Code, read.Body.String(), err)
	}
	if role.Seeded {
		t.Errorf("set_actions marked a hand-made role seeded; only the host's seeding call may")
	}

	add := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.read"}`)
	if add.Code != http.StatusCreated {
		t.Errorf("adding an action after set_actions answered %d %s, want 201", add.Code, add.Body.String())
	}
}
