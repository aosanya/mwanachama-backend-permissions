package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestARoleCodeThatOnlyLooksLikeAnotherIsRefusedAndAGrantMatchesTheCodeExactly(t *testing.T) {
	mux := mountPermissions(t)
	const check = "/permissions/check?subject=mallory&scope=instance:a&action=catalog.entry.delete"

	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.delete"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: add action answered %d %s", rec.Code, rec.Body.String())
	}

	for _, code := range []string{"Writer", "writer ", " writer", "wrіter", "a/b", "..", "writer%00"} {
		rec := send(t, mux, "POST", "/permissions/roles", `{"code":"`+code+`","name":"lookalike"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("role code %q answered %d %s, want 400 so no second role reads like writer", code, rec.Code, rec.Body.String())
		}
	}

	for _, code := range []string{"Writer", "WRITER", "writer "} {
		rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"mallory","scope":"instance:a","role_code":"`+code+`"}`)
		if rec.Code != http.StatusNotFound {
			t.Errorf("a grant naming role %q answered %d %s, want 404", code, rec.Code, rec.Body.String())
		}
	}
	if rec := send(t, mux, "GET", check, ""); strings.TrimSpace(rec.Body.String()) != "false" {
		t.Errorf("mallory holds no grant on writer yet the check answered %d %s, want false", rec.Code, rec.Body.String())
	}

	listed := send(t, mux, "GET", "/permissions/roles", "")
	if n := strings.Count(listed.Body.String(), `"code":`); n != 1 {
		t.Errorf("only writer should be declared, the listing holds %d roles: %s", n, listed.Body.String())
	}
}
