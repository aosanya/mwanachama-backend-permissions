package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const forgedClock = "1999-01-01T00:00:00Z"

func decodeInto(t *testing.T, step string, code int, want int, body []byte, into any) {
	t.Helper()
	if code != want {
		t.Fatalf("%s answered %d %s, want %d", step, code, body, want)
	}
	if err := json.Unmarshal(body, into); err != nil {
		t.Fatalf("%s answered %s: %v", step, body, err)
	}
}

func TestEveryWriteKeepsTheServersClockAndRetiredFlagOverTheBodys(t *testing.T) {
	mux := mountPermissions(t)

	type role struct {
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Deleted   bool   `json:"deleted"`
		Seeded    bool   `json:"seeded"`
	}
	var created role
	rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer","created_at":"`+forgedClock+`","updated_at":"`+forgedClock+`","deleted":true}`)
	decodeInto(t, "create role", rec.Code, http.StatusCreated, rec.Body.Bytes(), &created)
	if strings.HasPrefix(created.CreatedAt, "1999") || strings.HasPrefix(created.UpdatedAt, "1999") || created.Deleted {
		t.Errorf("create role kept the body's clock or retired flag: %s", rec.Body.String())
	}

	var patched role
	rec = send(t, mux, "PATCH", "/permissions/roles/writer", `{"name":"Renamed","created_at":"`+forgedClock+`","updated_at":"`+forgedClock+`"}`)
	decodeInto(t, "update role", rec.Code, http.StatusOK, rec.Body.Bytes(), &patched)
	var read role
	rec = send(t, mux, "GET", "/permissions/roles/writer", "")
	decodeInto(t, "read role", rec.Code, http.StatusOK, rec.Body.Bytes(), &read)
	if read.CreatedAt != created.CreatedAt || strings.HasPrefix(read.UpdatedAt, "1999") || read.Deleted {
		t.Errorf("update role moved created_at from %q or took the body's updated_at: %s", created.CreatedAt, rec.Body.String())
	}

	var action struct {
		CreatedAt string `json:"created_at"`
	}
	rec = send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert","created_at":"`+forgedClock+`"}`)
	decodeInto(t, "add action", rec.Code, http.StatusCreated, rec.Body.Bytes(), &action)
	if strings.HasPrefix(action.CreatedAt, "1999") {
		t.Errorf("add action kept the body's clock: %s", rec.Body.String())
	}

	rec = send(t, mux, "POST", "/permissions/grants", `{"subject":"member-a","scope":"instance:a","role_code":"writer","granted_at":"`+forgedClock+`","deleted":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create grant answered %d %s, want 201", rec.Code, rec.Body.String())
	}
	var grants []struct {
		GrantedAt string `json:"granted_at"`
		Deleted   bool   `json:"deleted"`
	}
	rec = send(t, mux, "GET", "/permissions/grants", "")
	decodeInto(t, "list grants", rec.Code, http.StatusOK, rec.Body.Bytes(), &grants)
	if len(grants) != 1 || strings.HasPrefix(grants[0].GrantedAt, "1999") || grants[0].Deleted {
		t.Errorf("create grant kept the body's granted_at or retired flag: %s", rec.Body.String())
	}
	if got := mayPerform(t, mux, "member-a", "instance:a", "catalog.entry.upsert"); got != "true" {
		t.Errorf("a grant created with a body saying deleted:true should still permit; the check answered %s", got)
	}
}
