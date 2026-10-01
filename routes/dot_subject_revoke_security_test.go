package routes_test

import (
	"net/http"
	"testing"
)

func TestAGrantWhoseSubjectIsADotSegmentCanStillBeWithdrawn(t *testing.T) {
	mux := mountPermissions(t)
	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.delete"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: add action answered %d %s", rec.Code, rec.Body.String())
	}
	cases := []struct{ subject, inPath, query string }{
		{"..", "%2E%2E", ".."},
		{".", "%2E", "."},
	}
	for _, c := range cases {
		if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"`+c.subject+`","scope":"module:catalog","role_code":"writer"}`); rec.Code != http.StatusCreated {
			t.Fatalf("precondition: grant to subject %q answered %d %s", c.subject, rec.Code, rec.Body.String())
		}
		check := "/permissions/check?subject=" + c.query + "&scope=module:catalog&action=catalog.entry.delete"
		if rec := send(t, mux, "GET", check, ""); rec.Code != http.StatusOK || rec.Body.String() != "true\n" && rec.Body.String() != "true" {
			t.Fatalf("precondition: check for subject %q answered %d %q, want true", c.subject, rec.Code, rec.Body.String())
		}
		if rec := send(t, mux, "DELETE", "/permissions/grants/"+c.inPath+"/module:catalog/writer", ""); rec.Code != http.StatusNoContent {
			t.Errorf("withdrawing the grant of subject %q through DELETE /grants/%s/… answered %d %s; want 204 — a grant the API accepted must be one the API can take back", c.subject, c.inPath, rec.Code, rec.Body.String())
		}
		if rec := send(t, mux, "GET", check, ""); rec.Body.String() != "false\n" && rec.Body.String() != "false" {
			t.Errorf("after withdrawal, check for subject %q answered %q; want false", c.subject, rec.Body.String())
		}
	}
}
