package routes_test

import (
	"encoding/json"
	"testing"
)

func TestPM16_OpenBug_ListGrantsIgnoresTheDeclaredRoleCodeQueryKey(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles", `{"code":"reader","name":"Reader"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"alice","scope":"module:catalog","role_code":"writer"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"bob","scope":"module:catalog","role_code":"reader"}`)

	rec := send(t, mux, "GET", "/permissions/grants?role_code=writer", "")
	if rec.Code != 200 {
		t.Fatalf("precondition: GET /grants?role_code=writer answered %d %s", rec.Code, rec.Body.String())
	}

	var grants []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &grants); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(grants) != 1 {
		t.Fatalf("PM16 open bug: GET /permissions/grants?role_code=writer returned %d grants (%s), want 1 — "+
			"the declared query key %q (the GrantFilter JSON tag, and the spelling list_grants' own response uses for each row) "+
			"does not bind, because dispatch's wholeQuery falls back to the lowercased Go field name %q when no `query` tag is set",
			len(grants), rec.Body.String(), "role_code", "rolecode")
	}
	if grants[0]["subject"] != "alice" {
		t.Fatalf("PM16 open bug: filtered-by-role_code listing returned %v, want alice's writer grant only", grants)
	}
}
