package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRenameRole_ABodySeededFalseDoesNotUnsealASeededRole(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"sealed","name":"Sealed"}`)
	if rec := send(t, mux, "PUT", "/permissions/roles/sealed/actions", `{"actions":["catalog.entry.list"]}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: set actions answered %d %s, want 200", rec.Code, rec.Body.String())
	}
	seeded := func() bool {
		t.Helper()
		read := send(t, mux, "GET", "/permissions/roles/sealed", "")
		var role struct {
			Seeded bool `json:"seeded"`
		}
		if err := json.Unmarshal(read.Body.Bytes(), &role); err != nil || read.Code != http.StatusOK {
			t.Fatalf("read role answered %d %s (%v)", read.Code, read.Body.String(), err)
		}
		return role.Seeded
	}
	if !seeded() {
		t.Fatalf("precondition: a role whose list a seeding pass replaced should read seeded=true")
	}

	if rec := send(t, mux, "PATCH", "/permissions/roles/sealed", `{"name":"Sealed again","seeded":false}`); rec.Code != http.StatusOK {
		t.Fatalf("renaming the seeded role answered %d %s, want 200", rec.Code, rec.Body.String())
	}

	if !seeded() {
		t.Errorf("a rename whose body said seeded=false unsealed a seeded role")
	}
	if add := send(t, mux, "POST", "/permissions/roles/sealed/actions", `{"action":"catalog.entry.delete"}`); add.Code != http.StatusConflict {
		t.Errorf("adding an action to a seeded role after a seeded=false rename answered %d %s, want 409", add.Code, add.Body.String())
	}
	if list := send(t, mux, "GET", "/permissions/roles/sealed/actions", ""); list.Code != http.StatusOK || strings.Contains(list.Body.String(), "catalog.entry.delete") {
		t.Errorf("the seeded role's list after the refused add answered %d %s, want 200 without catalog.entry.delete", list.Code, list.Body.String())
	}
}
