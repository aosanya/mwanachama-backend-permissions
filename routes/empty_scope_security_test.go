package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestAQuestionNamingNoScopeIsRefusedEvenForAnEverywhereGrant(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`)
	for _, body := range []string{
		`{"subject":"member-a","scope":"instance:a","role_code":"writer"}`,
		`{"subject":"member-b","scope":"*","role_code":"writer"}`,
	} {
		if rec := send(t, mux, "POST", "/permissions/grants", body); rec.Code != http.StatusCreated {
			t.Fatalf("precondition: create grant %s answered %d %s", body, rec.Code, rec.Body.String())
		}
	}
	if got := mayPerform(t, mux, "member-a", "instance:a", "catalog.entry.upsert"); got != "true" {
		t.Fatalf("precondition: member-a in its own scope answered %s, want true", got)
	}
	if got := mayPerform(t, mux, "member-b", "instance:z", "catalog.entry.upsert"); got != "true" {
		t.Fatalf("precondition: member-b's everywhere grant in instance:z answered %s, want true", got)
	}

	for _, subject := range []string{"member-a", "member-b"} {
		for _, path := range []string{
			"/permissions/check?subject=" + subject + "&scope=&action=catalog.entry.upsert",
			"/permissions/check?subject=" + subject + "&action=catalog.entry.upsert",
		} {
			rec := send(t, mux, "GET", path, "")
			if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "false" {
				t.Errorf("GET %s answered %d %s; a check that names no scope must answer false, never read an empty scope as everywhere", path, rec.Code, rec.Body.String())
			}
		}
		for _, path := range []string{
			"/permissions/subjects/" + subject + "/actions?scope=",
			"/permissions/subjects/" + subject + "/actions",
		} {
			rec := send(t, mux, "GET", path, "")
			if body := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || strings.Contains(body, "catalog.entry.upsert") {
				t.Errorf("GET %s answered %d %s; actions asked in no scope must list nothing", path, rec.Code, body)
			}
		}
	}
}
