package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestUpdateRole_ABodyCodeCannotRedirectTheRenameToAnotherRole(t *testing.T) {
	mux := mountPermissions(t)
	for _, body := range []string{`{"code":"writer","name":"Writer"}`, `{"code":"admin","name":"Admin"}`} {
		if rec := send(t, mux, "POST", "/permissions/roles", body); rec.Code != http.StatusCreated {
			t.Fatalf("precondition: create role %s answered %d %s", body, rec.Code, rec.Body.String())
		}
	}

	rec := send(t, mux, "PATCH", "/permissions/roles/writer", `{"code":"admin","name":"Renamed","seeded":true,"deleted":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /permissions/roles/writer answered %d %s, want 200", rec.Code, rec.Body.String())
	}

	read := func(code string) map[string]any {
		t.Helper()
		got := send(t, mux, "GET", "/permissions/roles/"+code, "")
		if got.Code != http.StatusOK {
			t.Fatalf("GET /permissions/roles/%s answered %d %s", code, got.Code, got.Body.String())
		}
		var role map[string]any
		if err := json.Unmarshal(got.Body.Bytes(), &role); err != nil {
			t.Fatalf("GET /permissions/roles/%s body %q: %v", code, got.Body.String(), err)
		}
		return role
	}

	if admin := read("admin"); admin["name"] != "Admin" {
		t.Errorf("a PATCH addressed to writer whose body named admin renamed admin to %v", admin["name"])
	}
	writer := read("writer")
	if writer["name"] != "Renamed" {
		t.Errorf("the role the path names was not the one renamed: writer reads %v", writer["name"])
	}
	if writer["seeded"] != false || writer["deleted"] != false {
		t.Errorf("an update body set server-owned state: seeded=%v deleted=%v", writer["seeded"], writer["deleted"])
	}
}
