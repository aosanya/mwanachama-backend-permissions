package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestPM7_OpenBug_ListRolesNameFilterIgnoresLikeWildcards(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles", `{"code":"reader","name":"Reader"}`)
	send(t, mux, "POST", "/permissions/roles", `{"code":"admin","name":"Admin"}`)

	for _, q := range []string{"_", "%"} {
		rec := send(t, mux, "GET", "/permissions/roles?name="+q, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /permissions/roles?name=%s answered %d %s", q, rec.Code, rec.Body.String())
		}
		var roles []struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &roles); err != nil {
			t.Fatalf("decode roles: %v", err)
		}
		if len(roles) != 0 {
			t.Errorf("PM7 (correctness) open bug: GET /permissions/roles?name=%s matched %d roles (%v); none of Writer, Reader or Admin contains the literal substring %q, so the filter should have matched none — %q is being read as a SQL LIKE wildcard instead of a literal character, the same unescaped-LIKE class AG32 already fixed in mwanachama-backend-agency's own likeFilter", q, len(roles), roles, q, q)
		}
	}
}
