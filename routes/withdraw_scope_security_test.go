package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestWithdrawGrant_AScopeCarryingASlashIsWithdrawnThroughTheRoute(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.list"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"alice","scope":"instance:abc/catalog","role_code":"writer"}`)

	check := "/permissions/check?subject=alice&scope=instance:abc/catalog&action=catalog.entry.list"
	if rec := send(t, mux, "GET", check, ""); strings.TrimSpace(rec.Body.String()) != "true" {
		t.Fatalf("precondition: the grant should permit the action before it is withdrawn, got %d %s", rec.Code, rec.Body.String())
	}

	rec := send(t, mux, "DELETE", "/permissions/grants/alice/instance:abc%2Fcatalog/writer", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("withdrawing a grant whose scope carries a slash answered %d %s, want 204", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "GET", check, ""); strings.TrimSpace(rec.Body.String()) != "false" {
		t.Errorf("a withdrawn grant still permits the action: %d %s", rec.Code, rec.Body.String())
	}
}
