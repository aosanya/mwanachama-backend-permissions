package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestWithdrawGrant_ASubjectCarryingASlashOrAnEscapeIsWithdrawnThroughTheRoute(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.list"}`)

	cases := []struct{ subject, checkQuery, deletePath string }{
		{"team/alice", "team%2Falice", "/permissions/grants/team%2Falice/instance:abc/writer"},
		{"a%2Fb", "a%252Fb", "/permissions/grants/a%252Fb/instance:abc/writer"},
	}
	for _, c := range cases {
		body := `{"subject":"` + c.subject + `","scope":"instance:abc","role_code":"writer"}`
		if rec := send(t, mux, "POST", "/permissions/grants", body); rec.Code != http.StatusCreated {
			t.Fatalf("precondition: granting subject %q answered %d %s, want 201", c.subject, rec.Code, rec.Body.String())
		}
		check := "/permissions/check?subject=" + c.checkQuery + "&scope=instance:abc&action=catalog.entry.list"
		if rec := send(t, mux, "GET", check, ""); strings.TrimSpace(rec.Body.String()) != "true" {
			t.Fatalf("precondition: subject %q should be permitted before withdrawal, got %d %s", c.subject, rec.Code, rec.Body.String())
		}
		if rec := send(t, mux, "DELETE", c.deletePath, ""); rec.Code != http.StatusNoContent {
			t.Fatalf("withdrawing subject %q answered %d %s, want 204", c.subject, rec.Code, rec.Body.String())
		}
		if rec := send(t, mux, "GET", check, ""); strings.TrimSpace(rec.Body.String()) != "false" {
			t.Errorf("subject %q is still permitted after its grant was withdrawn: %d %s", c.subject, rec.Code, rec.Body.String())
		}
	}
}
