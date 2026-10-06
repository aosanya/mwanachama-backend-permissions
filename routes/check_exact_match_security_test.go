package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestTheCheckAnswersOnlyTheExactSubjectAndScopeAGrantNames(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	if rec := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: add action answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"bob","scope":"module:catalog","role_code":"writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: grant answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "GET", "/permissions/check?subject=bob&scope=module:catalog&action=catalog.entry.upsert", ""); strings.TrimSpace(rec.Body.String()) != "true" {
		t.Fatalf("precondition: the grantee's own question answered %d %s, want true", rec.Code, rec.Body.String())
	}

	for _, query := range []string{
		"subject=BOB&scope=module:catalog&action=catalog.entry.upsert",
		"subject=Bob&scope=module:catalog&action=catalog.entry.upsert",
		"subject=bob%20&scope=module:catalog&action=catalog.entry.upsert",
		"subject=bo_&scope=module:catalog&action=catalog.entry.upsert",
		"subject=b%25&scope=module:catalog&action=catalog.entry.upsert",
		"subject=bob&scope=MODULE:CATALOG&action=catalog.entry.upsert",
		"subject=bob&scope=module:catalog%00&action=catalog.entry.upsert",
		"subject=bob&scope=module:catalo_&action=catalog.entry.upsert",
		"subject=bob&scope=module:%25&action=catalog.entry.upsert",
	} {
		rec := send(t, mux, "GET", "/permissions/check?"+query, "")
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "false" {
			t.Errorf("GET /permissions/check?%s answered %d %s, want 200 false: only the exact subject and scope a grant names may pass", query, rec.Code, rec.Body.String())
		}
	}

	for _, query := range []string{
		"subject=bob&scope=module:catalog&action=CATALOG.ENTRY.UPSERT",
		"subject=bob&scope=module:catalog&action=catalog.entry.%25",
		"subject=bob&scope=module:catalog&action=catalog.entry._psert",
	} {
		rec := send(t, mux, "GET", "/permissions/check?"+query, "")
		if strings.TrimSpace(rec.Body.String()) == "true" {
			t.Errorf("GET /permissions/check?%s answered %d %s, want a refusal or false: an action is matched exactly, never as a pattern", query, rec.Code, rec.Body.String())
		}
	}
}
