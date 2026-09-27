package routes_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

func newProbeServer(t *testing.T) *httptest.Server {
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
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestPM6_OpenBug_ListGrantsIgnoresRoleCodeQueryParam(t *testing.T) {
	srv := newProbeServer(t)
	client := srv.Client()

	mustPost := func(path, body string, want int) {
		t.Helper()
		resp, err := client.Post(srv.URL+path, "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != want {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("precondition: POST %s answered %d, want %d: %s", path, resp.StatusCode, want, b)
		}
	}

	mustPost("/permissions/roles", `{"code":"writer","name":"Writer"}`, http.StatusCreated)
	mustPost("/permissions/roles", `{"code":"reader","name":"Reader"}`, http.StatusCreated)
	mustPost("/permissions/grants",
		`{"subject":"alice","scope":"module:catalog","role_code":"writer"}`, http.StatusCreated)
	mustPost("/permissions/grants",
		`{"subject":"bob","scope":"module:catalog","role_code":"reader"}`, http.StatusCreated)

	resp, err := client.Get(srv.URL + "/permissions/grants?role_code=writer")
	if err != nil {
		t.Fatalf("GET grants filtered by role_code: %v", err)
	}
	defer resp.Body.Close()
	var grants []permissions.Grant
	if err := json.NewDecoder(resp.Body).Decode(&grants); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(grants) != 1 || grants[0].RoleCode != "writer" {
		t.Errorf("PM6 open bug: GET /permissions/grants?role_code=writer answered %d grant(s) %v, "+
			"want exactly the 1 grant for role_code=writer; the query filter uses the field name "+
			"the response body itself uses (role_code), and it is silently ignored, leaking every "+
			"other role's grants (here, bob's reader grant) into a filtered listing", len(grants), grants)
	}
}
