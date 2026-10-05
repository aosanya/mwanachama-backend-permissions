package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestGrantorClaimedInAnyKeySpellingNeverSurvivesWithOrWithoutAMountCaller(t *testing.T) {
	bodies := map[string]string{
		"exact":     `{"subject":"%s","scope":"module:catalog","role_code":"writer","granted_by":"eve"}`,
		"upper":     `{"subject":"%s","scope":"module:catalog","role_code":"writer","GRANTED_BY":"eve"}`,
		"mixed":     `{"subject":"%s","scope":"module:catalog","role_code":"writer","Granted_By":"eve"}`,
		"duplicate": `{"subject":"%s","scope":"module:catalog","role_code":"writer","granted_by":"eve","granted_by":"eve"}`,
	}
	for _, caller := range []string{"", "the-real-admin"} {
		mux := mountPermissionsAs(t, caller)
		if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); rec.Code != http.StatusCreated {
			t.Fatalf("precondition: create role answered %d %s", rec.Code, rec.Body.String())
		}
		for name, shape := range bodies {
			subject := "m-" + name
			body := fmt.Sprintf(shape, subject)
			if rec := send(t, mux, "POST", "/permissions/grants", body); rec.Code != http.StatusCreated {
				t.Fatalf("precondition: caller %q create grant %s answered %d %s", caller, body, rec.Code, rec.Body.String())
			}
			listed := send(t, mux, "GET", "/permissions/grants?subject="+subject, "")
			var grants []permissions.Grant
			if err := json.Unmarshal(listed.Body.Bytes(), &grants); err != nil || len(grants) != 1 {
				t.Fatalf("precondition: caller %q expected one stored grant for %s, got %d %s", caller, subject, listed.Code, listed.Body.String())
			}
			if grants[0].GrantedBy != caller {
				t.Errorf("caller %q, body %s: stored granted_by %q; it must be the mount's caller %q, never the body's claim",
					caller, body, grants[0].GrantedBy, caller)
			}
		}
	}
}
