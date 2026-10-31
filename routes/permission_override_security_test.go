package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAPermissionOverrideRecordsTheCallerNotTheBody(t *testing.T) {
	mux := mountPermissionsAs(t, "owner-1")
	rec := send(t, mux, "PUT", "/permissions/permissions/agy1/run_poll/treasurer", `{"held":true,"set_by":"someone-else","layer":"definition"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("override: %d %s", rec.Code, rec.Body)
	}
	rec = send(t, mux, "GET", "/permissions/permissions/agy1/run_poll/treasurer", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("read: %d %s", rec.Code, rec.Body)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["set_by"] != "owner-1" || got["layer"] != "override" || got["held"] != true {
		t.Fatalf("read back %v; want held, the override layer, set by the caller", got)
	}
	if rec := send(t, mux, "DELETE", "/permissions/permissions/agy1/run_poll/treasurer", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body)
	}
	if rec := send(t, mux, "GET", "/permissions/permissions/agy1/run_poll/treasurer", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("after clear: %d %s", rec.Code, rec.Body)
	}
}
