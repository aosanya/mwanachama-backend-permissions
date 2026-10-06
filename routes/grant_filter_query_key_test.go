package routes_test

import (
	"encoding/json"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestPM16_OpenBug_ListGrantsRoleCodeQueryParamIsIgnored(t *testing.T) {
	mux := mountPermissions(t)
	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != 201 {
		t.Fatalf("precondition: create writer role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"reader","name":"Reader"}`); rec.Code != 201 {
		t.Fatalf("precondition: create reader role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants",
		`{"subject":"alice","scope":"module:catalog","role_code":"writer"}`); rec.Code != 201 {
		t.Fatalf("precondition: grant alice writer answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants",
		`{"subject":"bob","scope":"module:catalog","role_code":"reader"}`); rec.Code != 201 {
		t.Fatalf("precondition: grant bob reader answered %d %s", rec.Code, rec.Body.String())
	}

	rec := send(t, mux, "GET", "/permissions/grants?role_code=writer", "")
	if rec.Code != 200 {
		t.Fatalf("GET /permissions/grants?role_code=writer answered %d %s", rec.Code, rec.Body.String())
	}
	var grants []permissions.Grant
	if err := json.Unmarshal(rec.Body.Bytes(), &grants); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(grants) != 1 || grants[0].RoleCode != "writer" {
		t.Errorf("PM16 open bug: GET /permissions/grants?role_code=writer returned %d grant(s) (%v), want exactly alice's "+
			"writer grant — role_code is the field every grant in the response is keyed by (GrantFilter's own json tag, "+
			"and the create-grant request body's own key), but GrantFilter.RoleCode carries no query tag so "+
			"dispatch.wholeQuery falls back to strings.ToLower(\"RoleCode\")=\"rolecode\", silently ignoring the "+
			"caller's role_code= filter and returning every live grant instead", len(grants), grants)
	}
}
