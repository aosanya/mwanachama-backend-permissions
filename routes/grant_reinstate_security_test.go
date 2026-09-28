package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

type reinstatingSubject struct{}

func TestReinstatedGrant_RecordsTheReinstatingCallerNotTheBodyOrThePreviousGrantor(t *testing.T) {
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
	mount := routes.Mount{Caller: func(ctx context.Context) string {
		held, _ := ctx.Value(reinstatingSubject{}).(string)
		return held
	}}
	for _, rt := range routes.RoutesFor(pm, mount) {
		mux.HandleFunc(rt.Pattern("/permissions"), rt.Handler)
	}
	as := func(subject, method, path, body string) (int, string) {
		t.Helper()
		signedIn := http.NewServeMux()
		signedIn.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), reinstatingSubject{}, subject)))
		}))
		rec := send(t, signedIn, method, path, body)
		return rec.Code, rec.Body.String()
	}

	as("alice", "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	if code, body := as("alice", "POST", "/permissions/grants", `{"subject":"bob","scope":"module:catalog","role_code":"writer"}`); code != http.StatusCreated {
		t.Fatalf("precondition: alice's grant answered %d %s", code, body)
	}
	if code, body := as("alice", "DELETE", "/permissions/grants/bob/module:catalog/writer", ""); code != http.StatusNoContent {
		t.Fatalf("precondition: withdraw answered %d %s", code, body)
	}

	code, body := as("mallory", "POST", "/permissions/grants", `{"subject":"bob","scope":"module:catalog","role_code":"writer","granted_by":"alice"}`)
	if code != http.StatusCreated {
		t.Fatalf("reinstating a withdrawn grant answered %d %s, want 201", code, body)
	}

	_, listed := as("mallory", "GET", "/permissions/grants?subject=bob", "")
	var grants []struct {
		GrantedBy string `json:"granted_by"`
	}
	if err := json.Unmarshal([]byte(listed), &grants); err != nil || len(grants) != 1 {
		t.Fatalf("listing bob's grants read %s (%v), want exactly one", listed, err)
	}
	if grants[0].GrantedBy != "mallory" {
		t.Errorf("a grant mallory reinstated reads granted_by=%q, want %q", grants[0].GrantedBy, "mallory")
	}
}
