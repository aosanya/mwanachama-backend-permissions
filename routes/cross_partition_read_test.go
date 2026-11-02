package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
	"github.com/aosanya/mwanachama-backend-permissions/models"
	"github.com/aosanya/mwanachama-backend-permissions/routes"
)

func partitionGatedPermissions(t *testing.T) (*http.ServeMux, permissions.PermissionsManager) {
	t.Helper()
	_, pm := mountPermissionsWithManager(t)
	mux := http.NewServeMux()
	for _, rt := range routes.Routes(pm) {
		handler, action := rt.Handler, rt.Action
		mux.HandleFunc(rt.Pattern("/permissions"), func(w http.ResponseWriter, r *http.Request) {
			held, err := pm.MayPerformIn(r.Context(), r.Header.Get("X-Caller"), r.Header.Get("X-Instance"), permissionsScope, action)
			if err != nil || !held {
				http.Error(w, "refused", http.StatusForbidden)
				return
			}
			handler(w, r)
		})
	}
	return mux, pm
}

func sendIn(mux *http.ServeMux, caller, instance, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("X-Caller", caller)
	req.Header.Set("X-Instance", instance)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func seedTwoAgencies(t *testing.T, pm permissions.PermissionsManager) {
	t.Helper()
	ctx := context.Background()
	roles := []struct {
		code    string
		actions []string
	}{
		{"keeper", []string{accessAction, "permissions.grant.list", "permissions.check.read"}},
		{"farmer", []string{"forms.answer.submit"}},
		{"auditor", []string{"audit.ledger.read"}},
	}
	for _, r := range roles {
		_, err := pm.CreateRole(ctx, models.Role{Code: r.code, Name: r.code})
		mustOK(t, "create role "+r.code, err)
		_, err = pm.SetActions(ctx, r.code, r.actions)
		mustOK(t, "set actions "+r.code, err)
	}
	for _, g := range []models.Grant{
		{Subject: "keeper-a", Scope: permissionsScope, Partition: "agency-a", RoleCode: "keeper"},
		{Subject: "asha", Scope: "module:forms", Partition: "agency-a", RoleCode: "farmer"},
		{Subject: "bola", Scope: "module:forms", Partition: "agency-b", RoleCode: "farmer"},
		{Subject: "bola", Scope: "module:audit", Partition: "agency-b", RoleCode: "auditor"},
	} {
		_, err := pm.CreateGrant(ctx, g)
		mustOK(t, "grant "+g.RoleCode+" to "+g.Subject, err)
	}
}

func subjectsOf(t *testing.T, body []byte) []string {
	t.Helper()
	var grants []models.Grant
	if err := json.Unmarshal(body, &grants); err != nil {
		t.Fatalf("answered %s: %v", body, err)
	}
	seen := map[string]bool{}
	for _, g := range grants {
		seen[g.Subject+"@"+g.Partition] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func TestAHolderInOneAgencyReadsAnotherAgencysAccessTodayBecauseTheGateSeesOnlyTheAction(t *testing.T) {
	mux, pm := partitionGatedPermissions(t)
	seedTwoAgencies(t, pm)

	rec := sendIn(mux, "keeper-a", "agency-a", "/permissions/subjects/bola/access?partition=agency-b")
	if rec.Code != http.StatusOK {
		t.Fatalf("keeper-a, holding the action in agency-a only, asked for bola in agency-b and was answered %d %s", rec.Code, rec.Body.String())
	}
	var lines []accessLine
	if err := json.Unmarshal(rec.Body.Bytes(), &lines); err != nil {
		t.Fatalf("answered %s: %v", rec.Body.String(), err)
	}
	actions := map[string]bool{}
	for _, l := range lines {
		actions[l.Action] = true
		if l.Partition != "agency-b" {
			t.Errorf("line %+v is not agency-b's", l)
		}
	}
	if !actions["forms.answer.submit"] || !actions["audit.ledger.read"] {
		t.Errorf("keeper-a read %v of bola in agency-b, want forms.answer.submit and audit.ledger.read: the read crosses the partition today", lines)
	}
}

func TestAHolderInOneAgencyListsOtherAgenciesGrantsTodayWithAPartitionOrWithNone(t *testing.T) {
	mux, pm := partitionGatedPermissions(t)
	seedTwoAgencies(t, pm)

	cases := map[string][]string{
		"/permissions/grants?partition=agency-b": {"bola@agency-b"},
		"/permissions/grants?partition=":         {"asha@agency-a", "bola@agency-b", "keeper-a@agency-a"},
		"/permissions/grants":                    {"asha@agency-a", "bola@agency-b", "keeper-a@agency-a"},
	}
	for path, want := range cases {
		rec := sendIn(mux, "keeper-a", "agency-a", path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s as keeper-a in agency-a answered %d %s", path, rec.Code, rec.Body.String())
			continue
		}
		got := subjectsOf(t, rec.Body.Bytes())
		if len(got) != len(want) {
			t.Errorf("GET %s listed %v, want %v: the list crosses the partition today", path, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("GET %s listed %v, want %v: the list crosses the partition today", path, got, want)
				break
			}
		}
	}
}

func TestAHolderInOneAgencyChecksAnotherAgencyTodayAndTheGateStillRefusesAskingFromIt(t *testing.T) {
	mux, pm := partitionGatedPermissions(t)
	seedTwoAgencies(t, pm)

	check := "/permissions/check?subject=bola&partition=agency-b&scope=module:forms&action=forms.answer.submit"
	rec := sendIn(mux, "keeper-a", "agency-a", check)
	if rec.Code != http.StatusOK || rec.Body.String() != "true\n" && rec.Body.String() != "true" {
		t.Errorf("keeper-a in agency-a checked bola in agency-b and was answered %d %q, want 200 true: the check crosses the partition today", rec.Code, rec.Body.String())
	}

	if rec := sendIn(mux, "keeper-a", "agency-b", check); rec.Code != http.StatusForbidden {
		t.Errorf("keeper-a presenting agency-b, where it holds nothing, was answered %d %s, want 403", rec.Code, rec.Body.String())
	}
}
