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

func mountPermissionsOver(t *testing.T) (*http.ServeMux, *gorm.DB) {
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
	return mux, db
}

func grantWriterTheUpsert(t *testing.T, mux *http.ServeMux) {
	t.Helper()
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	if rec := send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: add action answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "POST", "/permissions/grants", `{"subject":"bob","scope":"module:catalog","role_code":"writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: grant answered %d %s", rec.Code, rec.Body.String())
	}
	if got := mayPerform(t, mux, "bob", "module:catalog", "catalog.entry.upsert"); got != "true" {
		t.Fatalf("precondition: the grantee's question answered %s, want true", got)
	}
}

func TestTheCheckFailsClosedWhenItsStorageFails(t *testing.T) {
	for name, sabotage := range map[string]func(*testing.T, *gorm.DB){
		"connection closed": func(t *testing.T, db *gorm.DB) {
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatalf("db handle: %v", err)
			}
			if err := sqlDB.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
		},
		"every table gone": func(t *testing.T, db *gorm.DB) {
			tables, err := db.Migrator().GetTables()
			if err != nil {
				t.Fatalf("list tables: %v", err)
			}
			for _, table := range tables {
				if err := db.Migrator().DropTable(table); err != nil {
					t.Fatalf("drop %s: %v", table, err)
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			mux, db := mountPermissionsOver(t)
			grantWriterTheUpsert(t, mux)
			sabotage(t, db)

			check := send(t, mux, "GET", "/permissions/check?subject=bob&scope=module:catalog&action=catalog.entry.upsert", "")
			if check.Code == http.StatusOK || strings.TrimSpace(check.Body.String()) == "true" {
				t.Errorf("with %s, GET /permissions/check answered %d %s; want a non-200 refusal that never carries true", name, check.Code, check.Body.String())
			}
			listing := send(t, mux, "GET", "/permissions/subjects/bob/actions?scope=module:catalog", "")
			if listing.Code == http.StatusOK || strings.Contains(listing.Body.String(), "catalog.entry.upsert") {
				t.Errorf("with %s, GET /permissions/subjects/bob/actions answered %d %s; want a non-200 refusal naming no action", name, listing.Code, listing.Body.String())
			}
		})
	}
}
