package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func mayPerform(t *testing.T, mux *http.ServeMux, subject, scope, action string) string {
	t.Helper()
	rec := send(t, mux, "GET", "/permissions/check?subject="+subject+"&scope="+scope+"&action="+action, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("check answered %d %s", rec.Code, rec.Body.String())
	}
	return strings.TrimSpace(rec.Body.String())
}

func TestDeletedRoleStopsAuthorizingAndCannotBeRecreatedUnderItsCode(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"bob","scope":"module:catalog","role_code":"writer"}`)
	if got := mayPerform(t, mux, "bob", "module:catalog", "catalog.entry.upsert"); got != "true" {
		t.Fatalf("precondition: bob's grant authorizes, got %s", got)
	}

	if rec := send(t, mux, "DELETE", "/permissions/roles/writer", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("precondition: delete role answered %d %s", rec.Code, rec.Body.String())
	}
	if got := mayPerform(t, mux, "bob", "module:catalog", "catalog.entry.upsert"); got != "false" {
		t.Errorf("a grant of a deleted role still authorizes: check answered %s", got)
	}
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer again"}`)
	if got := mayPerform(t, mux, "bob", "module:catalog", "catalog.entry.upsert"); got != "false" {
		t.Errorf("recreating a deleted role's code revived its old grant: check answered %s", got)
	}
}

func TestGrantScopeMatchesExactlyAndAWithdrawnGrantStopsAtOnce(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"reader","name":"Reader"}`)
	send(t, mux, "POST", "/permissions/roles/reader/actions", `{"action":"x.y.z"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"dan","scope":"instance:a","role_code":"reader","deleted":true}`)
	if got := mayPerform(t, mux, "dan", "instance:a", "x.y.z"); got != "true" {
		t.Fatalf("precondition: dan's grant authorizes in instance:a, got %s", got)
	}
	for _, scope := range []string{"instance:A", "instance:a/catalog", "instance", "instance:ab"} {
		if got := mayPerform(t, mux, "dan", scope, "x.y.z"); got != "false" {
			t.Errorf("a grant in instance:a authorized scope %q: %s", scope, got)
		}
	}
	if rec := send(t, mux, "DELETE", "/permissions/grants/dan/instance:a/reader", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("precondition: withdraw answered %d %s", rec.Code, rec.Body.String())
	}
	if got := mayPerform(t, mux, "dan", "instance:a", "x.y.z"); got != "false" {
		t.Errorf("a withdrawn grant still authorizes: %s", got)
	}
}
