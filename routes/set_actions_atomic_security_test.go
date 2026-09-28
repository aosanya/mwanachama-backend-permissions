package routes_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSetActions_AFailedReplaceLeavesThePreviousListWhole(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "PUT", "/permissions/roles/writer/actions", `{"actions":["catalog.entry.list","catalog.entry.upsert"]}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"alice","scope":"instance:abc","role_code":"writer"}`)

	if rec := send(t, mux, "GET", "/permissions/check?subject=alice&scope=instance:abc&action=catalog.entry.upsert", ""); strings.TrimSpace(rec.Body.String()) != "true" {
		t.Fatalf("precondition: alice should hold catalog.entry.upsert, got %d %s", rec.Code, rec.Body.String())
	}

	rec := send(t, mux, "PUT", "/permissions/roles/writer/actions", `{"actions":["catalog.entry.list","catalog.entry.list"]}`)
	if rec.Code < 400 {
		t.Fatalf("a replace naming one action twice answered %d %s, want a refusal", rec.Code, rec.Body.String())
	}

	var held []struct {
		Action string `json:"action"`
	}
	list := send(t, mux, "GET", "/permissions/roles/writer/actions", "")
	if err := json.Unmarshal(list.Body.Bytes(), &held); err != nil {
		t.Fatalf("listing actions answered %d %s", list.Code, list.Body.String())
	}
	if len(held) != 2 {
		t.Errorf("after a refused replace the role holds %d actions %v, want the previous two untouched", len(held), held)
	}
	if rec := send(t, mux, "GET", "/permissions/check?subject=alice&scope=instance:abc&action=catalog.entry.upsert", ""); strings.TrimSpace(rec.Body.String()) != "true" {
		t.Errorf("after a refused replace alice lost catalog.entry.upsert: %d %s", rec.Code, rec.Body.String())
	}
}
