package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

func mountPermissions(t *testing.T) *http.ServeMux {
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
	return mux
}

func send(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestPM5_GrantedByIsNeverTakenFromTheRequestBody(t *testing.T) {
	mux := mountPermissions(t)
	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}

	rec := send(t, mux, "POST", "/permissions/grants",
		`{"subject":"mallory","scope":"module:catalog","role_code":"writer","granted_by":"root-admin"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create grant answered %d %s", rec.Code, rec.Body.String())
	}

	listed := send(t, mux, "GET", "/permissions/grants?subject=mallory", "")
	var grants []permissions.Grant
	if err := json.Unmarshal(listed.Body.Bytes(), &grants); err != nil || len(grants) != 1 {
		t.Fatalf("precondition: expected one stored grant, got %d %s", listed.Code, listed.Body.String())
	}
	if grants[0].GrantedBy == "root-admin" {
		t.Errorf("PM5 (P2) open hole: the stored grant's granted_by is %q, the value the request body claimed; "+
			"it should record the subject that actually made the grant, never a value the caller supplied", grants[0].GrantedBy)
	}
}
