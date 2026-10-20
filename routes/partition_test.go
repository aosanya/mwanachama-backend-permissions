package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestTheCheckAndTheWithdrawalAddressAGrantByItsPartition(t *testing.T) {
	mux := mountPermissions(t)
	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"answerer","name":"Answerer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/roles/answerer/actions", `{"action":"forms.answer.submit"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: add action answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"ann","scope":"module:forms","partition":"alpha","role_code":"answerer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: grant in alpha answered %d %s", rec.Code, rec.Body.String())
	}

	check := func(partition string) string {
		t.Helper()
		path := "/permissions/check?subject=ann&scope=module:forms&action=forms.answer.submit"
		if partition != "" {
			path += "&partition=" + partition
		}
		rec := send(t, mux, "GET", path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s answered %d %s", path, rec.Code, rec.Body.String())
		}
		return strings.TrimSpace(rec.Body.String())
	}
	if got := check("alpha"); got != "true" {
		t.Errorf("check in alpha answered %s, want true", got)
	}
	if got := check("beta"); got != "false" {
		t.Errorf("check in beta answered %s, want false: the grant names alpha", got)
	}
	if got := check(""); got != "false" {
		t.Errorf("check outside every partition answered %s, want false: the grant names alpha", got)
	}
	if rec := send(t, mux, "GET", "/permissions/check?subject=ann&scope=module:forms&action=forms.answer.submit&partition=al%20pha", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("a check naming a malformed partition answered %d, want 400", rec.Code)
	}

	if rec := send(t, mux, "DELETE", "/permissions/grants/ann/module:forms/answerer", ""); rec.Code != http.StatusNotFound {
		t.Errorf("withdrawing with no partition answered %d %s, want 404: the only grant names alpha", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "DELETE", "/permissions/grants/ann/module:forms/answerer?partition=alpha", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("withdrawing in alpha answered %d %s, want 204", rec.Code, rec.Body.String())
	}
	if got := check("alpha"); got != "false" {
		t.Errorf("after withdrawal, check in alpha answered %s, want false", got)
	}
}
