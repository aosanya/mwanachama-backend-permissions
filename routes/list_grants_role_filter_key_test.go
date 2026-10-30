package routes_test

import (
	"encoding/json"
	"testing"
)

func TestPM17_ListGrantsRoleFilterAnswersToRoleCode(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles", `{"code":"reader","name":"Reader"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"alice","scope":"*","role_code":"writer"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"bob","scope":"*","role_code":"reader"}`)

	rec := send(t, mux, "GET", "/permissions/grants?role_code=writer", "")
	if rec.Code != 200 {
		t.Fatalf("GET /grants?role_code=writer answered %d %s", rec.Code, rec.Body.String())
	}
	var rows []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 || rows[0]["subject"] != "alice" {
		t.Errorf("PM17 open bug: GET /permissions/grants?role_code=writer answered %d rows %v; want exactly alice's writer grant — GrantFilter.RoleCode carries no `query` tag, so the whole-query bind looks for the key `rolecode`, not `role_code`, and the misspelled filter silently narrows nothing instead of refusing or filtering", len(rows), rows)
	}
}
