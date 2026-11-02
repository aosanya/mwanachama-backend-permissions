package routes_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

const (
	accessAction     = "permissions.subject.access"
	permissionsScope = "module:permissions"
)

func gatedPermissions(t *testing.T) (*http.ServeMux, permissions.PermissionsManager) {
	t.Helper()
	_, pm := mountPermissionsWithManager(t)
	mux := http.NewServeMux()
	for _, rt := range routes.Routes(pm) {
		handler, action := rt.Handler, rt.Action
		mux.HandleFunc(rt.Pattern("/permissions"), func(w http.ResponseWriter, r *http.Request) {
			held, err := pm.MayPerform(r.Context(), r.Header.Get("X-Caller"), permissionsScope, action)
			if err != nil || !held {
				http.Error(w, "refused", http.StatusForbidden)
				return
			}
			handler(w, r)
		})
	}
	return mux, pm
}

func sendAs(mux *http.ServeMux, caller, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("X-Caller", caller)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

type accessLine struct {
	Action    string `json:"action"`
	RoleCode  string `json:"role_code"`
	RoleName  string `json:"role_name"`
	Scope     string `json:"scope"`
	Partition string `json:"partition"`
	GrantedBy string `json:"granted_by"`
	GrantedAt string `json:"granted_at"`
	EndsAt    string `json:"ends_at"`
}

func readAccess(t *testing.T, mux *http.ServeMux, subject, partition string) []accessLine {
	t.Helper()
	path := "/permissions/subjects/" + subject + "/access?partition=" + partition
	rec := sendAs(mux, "keeper", path)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s answered %d %s", path, rec.Code, rec.Body.String())
	}
	var lines []accessLine
	if err := json.Unmarshal(rec.Body.Bytes(), &lines); err != nil {
		t.Fatalf("GET %s answered %s: %v", path, rec.Body.String(), err)
	}
	return lines
}

func withoutTimes(t *testing.T, lines []accessLine) []accessLine {
	t.Helper()
	out := make([]accessLine, 0, len(lines))
	for _, l := range lines {
		if l.GrantedAt == "" {
			t.Errorf("line %+v carries no granted_at", l)
		}
		l.GrantedAt = ""
		out = append(out, l)
	}
	return out
}

func mustOK(t *testing.T, what string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("precondition: %s: %v", what, err)
	}
}

func seedAccess(t *testing.T, pm permissions.PermissionsManager) {
	t.Helper()
	ctx := context.Background()
	roles := []struct {
		code, name string
		actions    []string
	}{
		{"keeper", "Keeper", []string{accessAction}},
		{"farmer", "Farmer", []string{"forms.answer.submit", "shop.item.read"}},
		{"coordinator", "Coordinator", []string{"forms.answer.review", "forms.answer.submit"}},
		{"member", "Member", []string{"chat.thread.read", "shop.item.read"}},
		{"visitor", "Visitor", []string{"visit.page.read"}},
		{"temporary", "Temporary", []string{"temporary.thing.read"}},
		{"seasonal", "Seasonal", []string{"seasonal.thing.read"}},
	}
	for _, r := range roles {
		_, err := pm.CreateRole(ctx, models.Role{Code: r.code, Name: r.name})
		mustOK(t, "create role "+r.code, err)
		_, err = pm.SetActions(ctx, r.code, r.actions)
		mustOK(t, "set actions "+r.code, err)
	}

	soon := time.Now().Add(time.Second).Format(time.RFC3339Nano)
	grants := []models.Grant{
		{Subject: "keeper", Scope: permissionsScope, RoleCode: "keeper"},
		{Subject: "asha", Scope: "module:forms", Partition: "agency-a", RoleCode: "farmer", GrantedBy: "ruth"},
		{Subject: "asha", Scope: "module:forms", Partition: "agency-b", RoleCode: "farmer", GrantedBy: "ruth"},
		{Subject: "asha", Scope: "module:forms", Partition: "agency-b", RoleCode: "coordinator", GrantedBy: "ruth"},
		{Subject: "asha", Scope: "*", RoleCode: "member", GrantedBy: "ruth"},
		{Subject: "asha", Scope: "module:visits", Partition: "agency-a", RoleCode: "visitor", GrantedBy: "ruth"},
		{Subject: "asha", Scope: "module:temp", Partition: "agency-a", RoleCode: "temporary", GrantedBy: "ruth", EndsAt: soon},
		{Subject: "asha", Scope: "module:season", Partition: "agency-a", RoleCode: "seasonal", GrantedBy: "ruth", EndsAt: "2099-06-01T12:00:00Z"},
		{Subject: "grace", Scope: "module:forms", Partition: "agency-a", RoleCode: "coordinator", GrantedBy: "ruth"},
	}
	for _, g := range grants {
		_, err := pm.CreateGrant(ctx, g)
		mustOK(t, fmt.Sprintf("grant %s %s in %q", g.RoleCode, g.Subject, g.Partition), err)
	}
	mustOK(t, "retire visitor", pm.DeleteRole(ctx, "visitor"))
	time.Sleep(1200 * time.Millisecond)
}

func TestAccessOfReadsOnePartitionWithTheRoleBehindEachLine(t *testing.T) {
	mux, pm := gatedPermissions(t)
	seedAccess(t, pm)

	inA := readAccess(t, mux, "asha", "agency-a")
	wantA := []accessLine{
		{Action: "chat.thread.read", RoleCode: "member", RoleName: "Member", Scope: "*", GrantedBy: "ruth"},
		{Action: "forms.answer.submit", RoleCode: "farmer", RoleName: "Farmer", Scope: "module:forms", Partition: "agency-a", GrantedBy: "ruth"},
		{Action: "seasonal.thing.read", RoleCode: "seasonal", RoleName: "Seasonal", Scope: "module:season", Partition: "agency-a", GrantedBy: "ruth", EndsAt: "2099-06-01T12:00:00.000000000Z"},
		{Action: "shop.item.read", RoleCode: "farmer", RoleName: "Farmer", Scope: "module:forms", Partition: "agency-a", GrantedBy: "ruth"},
		{Action: "shop.item.read", RoleCode: "member", RoleName: "Member", Scope: "*", GrantedBy: "ruth"},
	}
	if got := withoutTimes(t, inA); !reflect.DeepEqual(got, wantA) {
		t.Errorf("asha in agency-a read\n%+v\nwant only agency-a's lines and the unpartitioned ones, without the retired role or the ended grant\n%+v", got, wantA)
	}

	inB := readAccess(t, mux, "asha", "agency-b")
	wantB := []accessLine{
		{Action: "chat.thread.read", RoleCode: "member", RoleName: "Member", Scope: "*", GrantedBy: "ruth"},
		{Action: "forms.answer.review", RoleCode: "coordinator", RoleName: "Coordinator", Scope: "module:forms", Partition: "agency-b", GrantedBy: "ruth"},
		{Action: "forms.answer.submit", RoleCode: "coordinator", RoleName: "Coordinator", Scope: "module:forms", Partition: "agency-b", GrantedBy: "ruth"},
		{Action: "forms.answer.submit", RoleCode: "farmer", RoleName: "Farmer", Scope: "module:forms", Partition: "agency-b", GrantedBy: "ruth"},
		{Action: "shop.item.read", RoleCode: "farmer", RoleName: "Farmer", Scope: "module:forms", Partition: "agency-b", GrantedBy: "ruth"},
		{Action: "shop.item.read", RoleCode: "member", RoleName: "Member", Scope: "*", GrantedBy: "ruth"},
	}
	if got := withoutTimes(t, inB); !reflect.DeepEqual(got, wantB) {
		t.Errorf("asha in agency-b read\n%+v\nwant only agency-b's lines and the unpartitioned ones\n%+v", got, wantB)
	}
}

func TestAccessOfInAPartitionTheSubjectHoldsNothingInShowsOnlyTheUnpartitionedLines(t *testing.T) {
	mux, pm := gatedPermissions(t)
	seedAccess(t, pm)

	got := withoutTimes(t, readAccess(t, mux, "asha", "agency-c"))
	want := []accessLine{
		{Action: "chat.thread.read", RoleCode: "member", RoleName: "Member", Scope: "*", GrantedBy: "ruth"},
		{Action: "shop.item.read", RoleCode: "member", RoleName: "Member", Scope: "*", GrantedBy: "ruth"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("asha in agency-c read\n%+v\nwant\n%+v", got, want)
	}
}

func TestAccessOfASubjectWithNoGrantsAnswersAnEmptyListNotNull(t *testing.T) {
	mux, pm := gatedPermissions(t)
	seedAccess(t, pm)

	rec := sendAs(mux, "keeper", "/permissions/subjects/nobody/access?partition=agency-a")
	if rec.Code != http.StatusOK {
		t.Fatalf("answered %d %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body != "[]\n" && body != "[]" {
		t.Errorf("a subject holding nothing answered %q, want []", body)
	}
}

func TestAccessOfNeverTruncatesAtTheListCap(t *testing.T) {
	mux, pm := gatedPermissions(t)
	ctx := context.Background()
	mustOK(t, "grant keeper", func() error {
		if _, err := pm.CreateRole(ctx, models.Role{Code: "keeper", Name: "Keeper"}); err != nil {
			return err
		}
		if _, err := pm.SetActions(ctx, "keeper", []string{accessAction}); err != nil {
			return err
		}
		_, err := pm.CreateGrant(ctx, models.Grant{Subject: "keeper", Scope: permissionsScope, RoleCode: "keeper"})
		return err
	}())

	const held = 250
	actions := make([]string, 0, held)
	for i := 0; i < held; i++ {
		actions = append(actions, fmt.Sprintf("bulk.thing.item_%03d", i))
	}
	_, err := pm.CreateRole(ctx, models.Role{Code: "bulk", Name: "Bulk"})
	mustOK(t, "create bulk", err)
	_, err = pm.SetActions(ctx, "bulk", actions)
	mustOK(t, "set bulk actions", err)
	_, err = pm.CreateGrant(ctx, models.Grant{Subject: "asha", Scope: "module:bulk", Partition: "agency-a", RoleCode: "bulk"})
	mustOK(t, "grant bulk", err)

	if got := len(readAccess(t, mux, "asha", "agency-a")); got != held {
		t.Errorf("a role holding %d actions read back %d lines, want all of them", held, got)
	}
}

func TestAccessOfIsRefusedToACallerWithoutItsAction(t *testing.T) {
	mux, pm := gatedPermissions(t)
	seedAccess(t, pm)
	ctx := context.Background()
	_, err := pm.CreateRole(ctx, models.Role{Code: "lister", Name: "Lister"})
	mustOK(t, "create lister", err)
	_, err = pm.SetActions(ctx, "lister", []string{"permissions.check.list", "permissions.grant.list", "permissions.role.read"})
	mustOK(t, "set lister actions", err)
	_, err = pm.CreateGrant(ctx, models.Grant{Subject: "lena", Scope: permissionsScope, RoleCode: "lister"})
	mustOK(t, "grant lister", err)

	path := "/permissions/subjects/asha/access?partition=agency-a"
	for _, caller := range []string{"", "mallory", "asha", "lena"} {
		if rec := sendAs(mux, caller, path); rec.Code != http.StatusForbidden {
			t.Errorf("caller %q read another person's access and was answered %d %s, want 403", caller, rec.Code, rec.Body.String())
		}
	}
	if rec := sendAs(mux, "keeper", path); rec.Code != http.StatusOK {
		t.Errorf("the caller holding %s was answered %d %s, want 200", accessAction, rec.Code, rec.Body.String())
	}
}

func TestAccessOfNeedsAWellFormedPartition(t *testing.T) {
	mux, pm := gatedPermissions(t)
	seedAccess(t, pm)

	for name, path := range map[string]string{
		"absent":    "/permissions/subjects/asha/access",
		"empty":     "/permissions/subjects/asha/access?partition=",
		"malformed": "/permissions/subjects/asha/access?partition=agency%20a",
	} {
		if rec := sendAs(mux, "keeper", path); rec.Code != http.StatusBadRequest {
			t.Errorf("a read with the partition %s answered %d %s, want 400: asking in no partition would hide which agency the lines belong to", name, rec.Code, rec.Body.String())
		}
	}
}

func TestAccessOfCarriesItsOwnActionAndNoOtherOperationDoes(t *testing.T) {
	built, err := routes.Build((*stubManager)(nil))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var carriers []string
	for _, r := range built {
		if r.Action == accessAction {
			carriers = append(carriers, r.Method+" "+r.Path)
		}
	}
	if want := []string{"GET /subjects/{subject}/access"}; !reflect.DeepEqual(carriers, want) {
		t.Errorf("routes carrying %s = %v, want %v", accessAction, carriers, want)
	}
}
