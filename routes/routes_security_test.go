package routes_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
)

type callerKey struct{}

func newSecurityManager(t *testing.T) permissions.PermissionsManager {
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
	m, err := permissions.NewPermissionsManager(db, s)
	if err != nil {
		t.Fatalf("NewPermissionsManager: %v", err)
	}
	return m
}

func serve(rs []routes.Route) http.Handler {
	mux := http.NewServeMux()
	for _, r := range rs {
		mux.HandleFunc(r.Pattern(""), r.Handler)
	}
	return mux
}

func call(h http.Handler, method, path, body, caller string) (int, string) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if caller != "" {
		req = req.WithContext(context.WithValue(req.Context(), callerKey{}, caller))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String())
}

func sentinelTable() map[string]error {
	return map[string]error{
		"ErrInvalidRole":    permissions.ErrInvalidRole,
		"ErrRoleNotFound":   permissions.ErrRoleNotFound,
		"ErrDuplicateRole":  permissions.ErrDuplicateRole,
		"ErrInvalidAction":  permissions.ErrInvalidAction,
		"ErrInvalidScope":   permissions.ErrInvalidScope,
		"ErrInvalidGrant":   permissions.ErrInvalidGrant,
		"ErrGrantNotFound":  permissions.ErrGrantNotFound,
		"ErrDuplicateGrant": permissions.ErrDuplicateGrant,
		"ErrSeededRole":     permissions.ErrSeededRole,
		"ErrNotPermitted":   permissions.ErrNotPermitted,
	}
}

func TestPM5_OpenHole_AGranterScopedToOneInstanceGrantsItselfEverywhere(t *testing.T) {
	m := newSecurityManager(t)
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("precondition: %v", err)
		}
	}
	_, err := m.CreateRole(ctx, permissions.Role{Code: "granter", Name: "Granter"})
	must(err)
	_, err = m.AddAction(ctx, "granter", "permissions.grant.create")
	must(err)
	_, err = m.CreateRole(ctx, permissions.Role{Code: "admin", Name: "Admin"})
	must(err)
	_, err = m.AddAction(ctx, "admin", "agency.goal.delete")
	must(err)
	_, err = m.CreateGrant(ctx, permissions.Grant{Subject: "mallory", Scope: "instance:a", RoleCode: "granter"})
	must(err)

	s, err := dispatch.Parse(permissions.Operations())
	must(err)
	gate := func(ctx context.Context, action string) error {
		subject, _ := ctx.Value(callerKey{}).(string)
		ok, err := m.MayPerform(ctx, subject, "instance:a", action)
		if err != nil || !ok {
			return errors.New("not permitted in instance:a")
		}
		return nil
	}
	rs, err := dispatch.Dispatch(s, dispatch.Deps{Manager: m, Errors: sentinelTable(), Authorize: gate})
	must(err)
	h := serve(rs)

	before, _ := m.MayPerform(ctx, "mallory", "instance:b", "agency.goal.delete")
	if before {
		t.Fatalf("precondition: mallory already holds agency.goal.delete in instance:b")
	}

	for _, scope := range []string{"*", "instance:b"} {
		code, body := call(h, "POST", "/grants", fmt.Sprintf(`{"subject":"mallory","scope":%q,"role_code":"admin"}`, scope), "mallory")
		if code == http.StatusCreated {
			t.Errorf("PM5 P1: mallory, holding permissions.grant.create only in instance:a, granted herself admin in scope %q through the gated route (got %d %s); a grant naming a scope the granter holds no grant.create in should be refused", scope, code, body)
		}
	}
	after, _ := m.MayPerform(ctx, "mallory", "instance:b", "agency.goal.delete")
	if after {
		t.Errorf("PM5 P1: after her own POST /grants, MayPerform(mallory, instance:b, agency.goal.delete) = true; she should hold nothing outside instance:a")
	}
}

func TestPM6_OpenHole_GrantedByIsTakenFromTheRequestBody(t *testing.T) {
	m := newSecurityManager(t)
	if _, err := m.CreateRole(context.Background(), permissions.Role{Code: "editor", Name: "Editor"}); err != nil {
		t.Fatalf("precondition: %v", err)
	}
	h := serve(routes.Routes(m))

	code, body := call(h, "POST", "/grants", `{"subject":"alice","scope":"instance:a","role_code":"editor","granted_by":"the-real-owner"}`, "mallory")
	if code != http.StatusCreated && code != http.StatusBadRequest {
		t.Fatalf("precondition: POST /grants answered %d %s", code, body)
	}
	grants, err := m.ListGrants(context.Background(), permissions.GrantFilter{Subject: "alice"})
	if err != nil {
		t.Fatalf("ListGrants: %v", err)
	}
	for _, g := range grants {
		if g.GrantedBy == "the-real-owner" {
			t.Errorf("PM6 P1: POST /grants stored granted_by %q exactly as the request body claimed it (response %d %s); the trail field should be set from the authenticated caller or refused, never taken from the body", g.GrantedBy, code, body)
		}
	}
}

func TestPM7_OpenHole_SeededIsTakenFromTheRequestBody(t *testing.T) {
	m := newSecurityManager(t)
	h := serve(routes.Routes(m))

	code, body := call(h, "POST", "/roles", `{"code":"handmade","name":"Hand made","seeded":true}`, "")
	if code != http.StatusCreated && code != http.StatusBadRequest {
		t.Fatalf("precondition: POST /roles answered %d %s", code, body)
	}
	role, err := m.GetRole(context.Background(), "handmade")
	if err == nil && role.Seeded {
		t.Errorf("PM7 P3: POST /roles stored seeded=true from the request body (response %d %s); only a seeding pass should mark a role seeded, a hand-made role must start unseeded", code, body)
	}
}

func TestPM8_OpenHole_GrantListingHasNoCap(t *testing.T) {
	m := newSecurityManager(t)
	ctx := context.Background()
	if _, err := m.CreateRole(ctx, permissions.Role{Code: "reader", Name: "Reader"}); err != nil {
		t.Fatalf("precondition: %v", err)
	}
	const planted = 1200
	for i := 0; i < planted; i++ {
		if _, err := m.CreateGrant(ctx, permissions.Grant{Subject: fmt.Sprintf("s%04d", i), Scope: "*", RoleCode: "reader"}); err != nil {
			t.Fatalf("precondition: grant %d: %v", i, err)
		}
	}
	h := serve(routes.Routes(m))

	code, body := call(h, "GET", "/grants?limit=5", "", "")
	returned := strings.Count(body, `"subject"`)
	if code == http.StatusOK && returned >= planted {
		t.Errorf("PM8 P3: GET /grants?limit=5 returned all %d of %d grants (%d bytes); a listing should be capped by default and honour a caller's smaller limit", returned, planted, len(body))
	}
}

func TestGrantWithASlashInItsScopeCanBeWithdrawnOverHTTP(t *testing.T) {
	m := newSecurityManager(t)
	h := serve(routes.Routes(m))
	if code, body := call(h, "POST", "/roles", `{"code":"editor","name":"Editor"}`, ""); code != http.StatusCreated {
		t.Fatalf("precondition: create role %d %s", code, body)
	}
	if code, body := call(h, "POST", "/grants", `{"subject":"bob","scope":"instance:a/b","role_code":"editor"}`, ""); code != http.StatusCreated {
		t.Fatalf("precondition: create grant %d %s", code, body)
	}
	if code, body := call(h, "DELETE", "/grants/bob/instance:a%2Fb/editor", "", ""); code != http.StatusNoContent {
		t.Fatalf("withdrawing a grant whose scope carries a slash answered %d %s; it must stay revocable over HTTP", code, body)
	}
	ok, err := m.MayPerform(context.Background(), "bob", "instance:a/b", "agency.goal.create")
	if err != nil || ok {
		t.Fatalf("after withdrawal MayPerform = %v, %v", ok, err)
	}
	grants, _ := m.ListGrants(context.Background(), permissions.GrantFilter{Subject: "bob"})
	if len(grants) != 0 {
		t.Fatalf("the withdrawn grant still lists: %+v", grants)
	}
}
