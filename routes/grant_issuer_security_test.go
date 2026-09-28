package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

type subjectKey struct{}

func mountPermissionsAs(t *testing.T, subject string) *http.ServeMux {
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

	mount := routes.Mount{Caller: func(ctx context.Context) string {
		held, _ := ctx.Value(subjectKey{}).(string)
		return held
	}}
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(pm, mount) {
		mux.HandleFunc(rt.Pattern("/permissions"), rt.Handler)
	}

	signedIn := http.NewServeMux()
	signedIn.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), subjectKey{}, subject)))
	}))
	return signedIn
}

func TestPM5_TheMountSuppliesTheGrantor(t *testing.T) {
	const subject = "the-real-admin"
	const claimed = "root-admin"
	mux := mountPermissionsAs(t, subject)

	if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
	}
	rec := send(t, mux, "POST", "/permissions/grants",
		`{"subject":"mallory","scope":"module:catalog","role_code":"writer","granted_by":"`+claimed+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("precondition: create grant answered %d %s", rec.Code, rec.Body.String())
	}

	listed := send(t, mux, "GET", "/permissions/grants?subject=mallory", "")
	var grants []permissions.Grant
	if err := json.Unmarshal(listed.Body.Bytes(), &grants); err != nil || len(grants) != 1 {
		t.Fatalf("precondition: expected one stored grant, got %d %s", listed.Code, listed.Body.String())
	}
	if grants[0].GrantedBy != subject {
		t.Errorf("the stored grantor is %q; it must be the caller %q the mount supplied, never the body's %q",
			grants[0].GrantedBy, subject, claimed)
	}
}

func TestPM5_AGrantorArgumentIsNeverOfferedToAnAgent(t *testing.T) {
	s, err := dispatch.Parse(permissions.Operations())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	op, ok := s.Operations["create_grant"]
	if !ok {
		t.Fatal("the spec declares no create_grant")
	}
	for _, a := range op.Args {
		if a.Into == "GrantedBy" && a.From != dispatch.FromCaller {
			t.Errorf("create_grant binds GrantedBy from %q; only the caller may supply a grantor", a.From)
		}
	}
}
