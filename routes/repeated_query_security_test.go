package routes_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRepeatedQueryArguments_TheCheckAndTheListingReadTheSameValue(t *testing.T) {
	mux := mountPermissions(t)
	send(t, mux, "POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`)
	send(t, mux, "POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.list"}`)
	send(t, mux, "POST", "/permissions/grants", `{"subject":"alice","scope":"instance:abc","role_code":"writer"}`)

	if rec := send(t, mux, "GET", "/permissions/check?subject=alice&scope=instance:abc&action=catalog.entry.list", ""); strings.TrimSpace(rec.Body.String()) != "true" {
		t.Fatalf("precondition: alice should be permitted in instance:abc, got %d %s", rec.Code, rec.Body.String())
	}

	for _, order := range [][2]string{{"alice", "bob"}, {"bob", "alice"}} {
		check := send(t, mux, "GET", "/permissions/check?subject="+order[0]+"&subject="+order[1]+"&scope=instance:abc&action=catalog.entry.list", "")
		list := send(t, mux, "GET", "/permissions/grants?subject="+order[0]+"&subject="+order[1], "")
		var grants []struct {
			Subject string `json:"subject"`
		}
		if err := json.Unmarshal(list.Body.Bytes(), &grants); err != nil {
			t.Fatalf("listing with subjects %v answered %d %s", order, list.Code, list.Body.String())
		}
		permitted := strings.TrimSpace(check.Body.String()) == "true"
		listed := len(grants) > 0
		if permitted != listed {
			t.Errorf("subjects %v: the check answered permitted=%v while the listing showed a grant=%v — the two read different values of a repeated subject", order, permitted, listed)
		}
		for _, g := range grants {
			if g.Subject != order[0] {
				t.Errorf("subjects %v: listing returned %q, want only the first value %q", order, g.Subject, order[0])
			}
		}
	}
}
