package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

func mountPermissionsWithManager(t *testing.T) (*http.ServeMux, permissions.PermissionsManager) {
	t.Helper()
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
	for _, rt := range routes.Routes(pm) {
		mux.HandleFunc(rt.Pattern("/permissions"), rt.Handler)
	}
	return mux, pm
}

func TestRenameRole_ABodySeededFalseDoesNotUnsealASeededRole(t *testing.T) {
	mux, pm := mountPermissionsWithManager(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"sealed","name":"Sealed"}`)
	if _, err := pm.SeedActions(context.Background(), "sealed", []string{"catalog.entry.list"}); err != nil {
		t.Fatalf("precondition: seed actions: %v", err)
	}
	seeded := func() bool {
		t.Helper()
		read := send(t, mux, "GET", "/permissions/roles/sealed", "")
		var role struct {
			Seeded bool `json:"seeded"`
		}
		if err := json.Unmarshal(read.Body.Bytes(), &role); err != nil || read.Code != http.StatusOK {
			t.Fatalf("read role answered %d %s (%v)", read.Code, read.Body.String(), err)
		}
		return role.Seeded
	}
	if !seeded() {
		t.Fatalf("precondition: a role whose list a seeding pass replaced should read seeded=true")
	}

	if rec := send(t, mux, "PATCH", "/permissions/roles/sealed", `{"name":"Sealed again","seeded":false}`); rec.Code != http.StatusOK {
		t.Fatalf("renaming the seeded role answered %d %s, want 200", rec.Code, rec.Body.String())
	}

	if !seeded() {
		t.Errorf("a rename whose body said seeded=false unsealed a seeded role")
	}
	if add := send(t, mux, "POST", "/permissions/roles/sealed/actions", `{"action":"catalog.entry.delete"}`); add.Code != http.StatusConflict {
		t.Errorf("adding an action to a seeded role after a seeded=false rename answered %d %s, want 409", add.Code, add.Body.String())
	}
	if list := send(t, mux, "GET", "/permissions/roles/sealed/actions", ""); list.Code != http.StatusOK || strings.Contains(list.Body.String(), "catalog.entry.delete") {
		t.Errorf("the seeded role's list after the refused add answered %d %s, want 200 without catalog.entry.delete", list.Code, list.Body.String())
	}
}
