package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const forgedClock = "1999-01-01T00:00:00Z"

func TestAWriteKeepsTheServersClockAndLivenessWhateverTheBodyClaims(t *testing.T) {
	mux := mountPermissions(t)

	role := send(t, mux, "POST", "/permissions/roles",
		`{"code":"writer","name":"Writer","created_at":"`+forgedClock+`","updated_at":"`+forgedClock+`","deleted":true}`)
	if role.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", role.Code, role.Body.String())
	}
	var made struct {
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Deleted   bool   `json:"deleted"`
	}
	if err := json.Unmarshal(role.Body.Bytes(), &made); err != nil {
		t.Fatalf("create role body: %v", err)
	}
	if made.CreatedAt == forgedClock || made.UpdatedAt == forgedClock || made.Deleted {
		t.Errorf("POST /roles kept a body clock or liveness: %s", role.Body.String())
	}

	send(t, mux, "POST", "/permissions/roles", `{"code":"reader","name":"Reader"}`)
	added := send(t, mux, "POST", "/permissions/roles/reader/actions",
		`{"action":"catalog.entry.upsert","created_at":"`+forgedClock+`","role_code":"writer"}`)
	if added.Code != http.StatusCreated {
		t.Fatalf("precondition: add action answered %d %s", added.Code, added.Body.String())
	}
	if strings.Contains(added.Body.String(), forgedClock) {
		t.Errorf("POST /roles/reader/actions kept the body clock: %s", added.Body.String())
	}
	if listed := send(t, mux, "GET", "/permissions/roles/writer/actions", ""); strings.TrimSpace(listed.Body.String()) != "[]" {
		t.Errorf("a body role_code moved the action onto the role the path did not name: %s", listed.Body.String())
	}

	grant := send(t, mux, "POST", "/permissions/grants",
		`{"subject":"bob","scope":"*","role_code":"reader","granted_at":"`+forgedClock+`","deleted":true}`)
	if grant.Code != http.StatusCreated {
		t.Fatalf("precondition: create grant answered %d %s", grant.Code, grant.Body.String())
	}
	if strings.Contains(grant.Body.String(), forgedClock) || strings.Contains(grant.Body.String(), `"deleted":true`) {
		t.Errorf("POST /grants kept a body clock or liveness: %s", grant.Body.String())
	}
	if listed := send(t, mux, "GET", "/permissions/grants", ""); strings.Contains(listed.Body.String(), forgedClock) {
		t.Errorf("a stored grant carries the body's clock: %s", listed.Body.String())
	}
}
