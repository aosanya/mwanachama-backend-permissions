package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestAnEmptySubjectCanNeitherHoldAGrantNorPassTheCheck(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	if rec := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: add action answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"bob","scope":"*","role_code":"writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: a named subject's everywhere grant answered %d %s", rec.Code, rec.Body.String())
	}

	for _, body := range []string{
		`{"subject":"","scope":"*","role_code":"writer"}`,
		`{"scope":"*","role_code":"writer"}`,
	} {
		if rec := send(t, mux, "POST", "/permissions/grants", body); rec.Code != http.StatusBadRequest {
			t.Errorf("POST /grants %s answered %d %s, want 400", body, rec.Code, rec.Body.String())
		}
	}

	for _, path := range []string{
		"/permissions/check?subject=&scope=module:catalog&action=catalog.entry.upsert",
		"/permissions/check?scope=module:catalog&action=catalog.entry.upsert",
	} {
		rec := send(t, mux, "GET", path, "")
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "false" {
			t.Errorf("GET %s answered %d %s, want 200 false", path, rec.Code, rec.Body.String())
		}
	}

	listed := send(t, mux, "GET", "/permissions/grants", "")
	if strings.Count(listed.Body.String(), `"subject"`) != 1 {
		t.Errorf("a grant to an empty subject was stored: %s", listed.Body.String())
	}
}
