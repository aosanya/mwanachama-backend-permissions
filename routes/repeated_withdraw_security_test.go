package routes_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

func TestRepeatedWithdrawAndRetireNeverReviveAGrantForTheCheckOrTheListing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := permissions.LoadSpec("../spec/examples/wakala.permissions.json")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if err := permissions.Migrate(db, s); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	pm, err := permissions.NewPermissionsManager(db, s)
	if err != nil {
		t.Fatalf("NewPermissionsManager: %v", err)
	}
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(pm, routes.Mount{}) {
		mux.HandleFunc(rt.Pattern("/permissions"), rt.Handler)
	}
	call := func(method, path, body string) (int, string) {
		t.Helper()
		rec := send(t, mux, method, path, body)
		return rec.Code, strings.TrimSpace(rec.Body.String())
	}
	const check = "/permissions/check?subject=bob&scope=module:catalog&action=catalog.entry.upsert"

	call("POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	call("POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`)
	if code, body := call("POST", "/permissions/grants", `{"subject":"bob","scope":"module:catalog","role_code":"writer"}`); code != http.StatusCreated {
		t.Fatalf("precondition: the grant answered %d %s", code, body)
	}
	if code, body := call("GET", check, ""); code != http.StatusOK || body != "true" {
		t.Fatalf("precondition: the check before any withdraw answered %d %s; want 200 true", code, body)
	}

	for i := 0; i < 3; i++ {
		if code, body := call("DELETE", "/permissions/grants/bob/module:catalog/writer", ""); code != http.StatusNoContent {
			t.Fatalf("withdraw #%d answered %d %s; want 204", i+1, code, body)
		}
		if code, body := call("GET", check, ""); body != "false" {
			t.Errorf("after withdraw #%d the check answered %d %s; want false", i+1, code, body)
		}
		if _, body := call("GET", "/permissions/grants?subject=bob", ""); body != "[]" {
			t.Errorf("after withdraw #%d the listing answered %s; want []", i+1, body)
		}
	}

	if code, body := call("POST", "/permissions/grants", `{"subject":"bob","scope":"module:catalog","role_code":"writer"}`); code != http.StatusCreated {
		t.Fatalf("precondition: re-granting answered %d %s", code, body)
	}
	for i := 0; i < 2; i++ {
		if code, body := call("DELETE", "/permissions/roles/writer", ""); code != http.StatusNoContent {
			t.Fatalf("retiring the role #%d answered %d %s; want 204", i+1, code, body)
		}
		if code, body := call("DELETE", "/permissions/roles/writer/actions/catalog.entry.upsert", ""); code != http.StatusNoContent {
			t.Fatalf("removing the action from the retired role #%d answered %d %s; want 204", i+1, code, body)
		}
		if code, body := call("GET", check, ""); body != "false" {
			t.Errorf("after retiring the role #%d the check answered %d %s; want false", i+1, code, body)
		}
		if _, body := call("GET", "/permissions/subjects/bob/actions?scope=module:catalog", ""); body != "[]" && body != "null" {
			t.Errorf("after retiring the role #%d bob's actions answered %s; want none", i+1, body)
		}
	}
}
