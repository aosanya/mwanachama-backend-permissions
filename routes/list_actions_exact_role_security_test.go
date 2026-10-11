package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestListActionsAnswersOnlyTheRoleItsPathNames(t *testing.T) {
	mux := mountPermissions(t)
	for _, r := range []struct{ path, body string }{
		{"/permissions/roles", `{"code":"writer","name":"Writer"}`},
		{"/permissions/roles/writer/actions", `{"action":"catalog.entry.list"}`},
		{"/permissions/roles", `{"code":"reader","name":"Reader"}`},
		{"/permissions/roles/reader/actions", `{"action":"catalog.entry.open"}`},
	} {
		if rec := send(t, mux, "POST", r.path, r.body); rec.Code != http.StatusCreated {
			t.Fatalf("precondition: POST %s answered %d %s, want 201", r.path, rec.Code, rec.Body.String())
		}
	}
	own := send(t, mux, "GET", "/permissions/roles/writer/actions", "")
	if own.Code != http.StatusOK || !strings.Contains(own.Body.String(), `"catalog.entry.list"`) {
		t.Fatalf("precondition: writer's own actions answered %d %s, want 200 naming catalog.entry.list", own.Code, own.Body.String())
	}

	for _, path := range []string{
		"/permissions/roles/%25/actions",
		"/permissions/roles/wri_er/actions",
		"/permissions/roles/_eader/actions",
		"/permissions/roles/WRITER/actions",
		"/permissions/roles/writer%25/actions",
	} {
		rec := send(t, mux, "GET", path, "")
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
			t.Errorf("GET %s answered %d %s, want 200 [] — a role code is matched exactly, never as a pattern", path, rec.Code, rec.Body.String())
		}
	}

	steered := send(t, mux, "GET", "/permissions/roles/writer/actions?role_code=reader&roleCode=reader&rolecode=reader", "")
	if strings.Contains(steered.Body.String(), "catalog.entry.open") || !strings.Contains(steered.Body.String(), "catalog.entry.list") {
		t.Errorf("a query role_code steered the listing off its path: %d %s, want writer's actions only", steered.Code, steered.Body.String())
	}
}
