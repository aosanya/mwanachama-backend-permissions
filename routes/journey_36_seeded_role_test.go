package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

type journey36Szentendre struct {
	mux *http.ServeMux
	pm  permissions.PermissionsManager
}

func journey36SzentendrePermissions(t *testing.T) journey36Szentendre {
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
	ctx := context.Background()
	if _, err := pm.CreateRole(ctx, permissions.Role{Code: "district_leader", Name: "District leader"}); err != nil {
		t.Fatalf("precondition: District leader: %v", err)
	}
	if _, err := pm.SeedActions(ctx, "district_leader", []string{"forms.form.read", "comm.chat.post"}); err != nil {
		t.Fatalf("precondition: the template's publish seeds District leader: %v", err)
	}
	return journey36Szentendre{mux: mux, pm: pm}
}

func (z journey36Szentendre) seeded(t *testing.T, code string) bool {
	t.Helper()
	rec := send(t, z.mux, "GET", "/permissions/roles/"+code, "")
	var role struct {
		Seeded bool `json:"seeded"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &role); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("read %s answered %d %s", code, rec.Code, rec.Body.String())
	}
	return role.Seeded
}

func (z journey36Szentendre) actions(t *testing.T, code string) []string {
	t.Helper()
	rec := send(t, z.mux, "GET", "/permissions/roles/"+code+"/actions", "")
	var items []struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		var wrapped struct {
			Items []struct {
				Action string `json:"action"`
			} `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &wrapped); err != nil {
			t.Fatalf("list %s actions answered %d %s", code, rec.Code, rec.Body.String())
		}
		items = wrapped.Items
	}
	var out []string
	for _, a := range items {
		out = append(out, a.Action)
	}
	sort.Strings(out)
	return out
}

func TestJourney36_19_OnlyPublishingATemplatesRolesMarksARoleSeeded(t *testing.T) {
	z := journey36SzentendrePermissions(t)

	if rec := send(t, z.mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer","seeded":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("Katalin B creating Writer answered %d %s, want 201", rec.Code, rec.Body.String())
	}
	if z.seeded(t, "writer") {
		t.Errorf("Writer reads seeded=true; the body's seeded must be ignored (decision 444)")
	}
	if rec := send(t, z.mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`); rec.Code != http.StatusCreated {
		t.Errorf("adding an action to Writer answered %d %s, want 201", rec.Code, rec.Body.String())
	}
	if rec := send(t, z.mux, "DELETE", "/permissions/roles/writer/actions/catalog.entry.upsert", ""); rec.Code != http.StatusNoContent {
		t.Errorf("removing Writer's action answered %d %s, want 204", rec.Code, rec.Body.String())
	}
	if rec := send(t, z.mux, "PUT", "/permissions/roles/writer/actions", `{"actions":["catalog.entry.read"]}`); rec.Code != http.StatusOK {
		t.Errorf("replacing Writer's actions answered %d %s, want 200", rec.Code, rec.Body.String())
	}
	if z.seeded(t, "writer") {
		t.Errorf("replacing Writer's actions marked it seeded (decision 444)")
	}

	rec := send(t, z.mux, "GET", "/permissions/roles?seeded=true", "")
	var listed []struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("the seeded listing answered %d %s", rec.Code, rec.Body.String())
	}
	var codes []string
	for _, r := range listed {
		codes = append(codes, r.Code)
	}
	if len(codes) != 1 || codes[0] != "district_leader" {
		t.Errorf("the seeded listing names %v; want District leader and never Writer (decision 444)", codes)
	}

	if rec := send(t, z.mux, "POST", "/permissions/roles/district_leader/actions", `{"action":"forms.form.send"}`); rec.Code != http.StatusConflict {
		t.Errorf("adding an action to the seeded District leader answered %d %s, want 409 as a seeded role's", rec.Code, rec.Body.String())
	}
	if rec := send(t, z.mux, "DELETE", "/permissions/roles/district_leader/actions/comm.chat.post", ""); rec.Code != http.StatusConflict {
		t.Errorf("removing an action from the seeded District leader answered %d %s, want 409 as a seeded role's", rec.Code, rec.Body.String())
	}
	if rec := send(t, z.mux, "PUT", "/permissions/roles/district_leader/actions", `{"actions":["forms.form.read"]}`); rec.Code != http.StatusOK {
		t.Errorf("replacing District leader's actions answered %d %s, want 200", rec.Code, rec.Body.String())
	}
	if got := z.actions(t, "district_leader"); len(got) != 1 || got[0] != "forms.form.read" {
		t.Errorf("District leader reads back %v after the replace; want [forms.form.read]", got)
	}
	if !z.seeded(t, "district_leader") {
		t.Errorf("replacing District leader's actions cleared its seeded mark; it must stay seeded (decision 444)")
	}
}
