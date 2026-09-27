package routes_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestPM7_OpenBug_SetActionsWithoutActionsFieldWipesAndLocksTheRole(t *testing.T) {
	srv := newProbeServer(t)
	client := srv.Client()

	do := func(method, path, body string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("build %s %s: %v", method, path, err)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		return resp
	}

	if resp := do("POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("precondition: create role answered %d %s", resp.StatusCode, b)
	}
	addResp := do("POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.read"}`)
	if addResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(addResp.Body)
		addResp.Body.Close()
		t.Fatalf("precondition: add action answered %d %s", addResp.StatusCode, b)
	}
	addResp.Body.Close()

	wipe := do("PUT", "/permissions/roles/writer/actions", `{}`)
	wipeBody, _ := io.ReadAll(wipe.Body)
	wipe.Body.Close()

	after := do("GET", "/permissions/roles/writer/actions", "")
	var actions []permissions.RoleAction
	if err := json.NewDecoder(after.Body).Decode(&actions); err != nil {
		t.Fatalf("decode actions: %v", err)
	}
	after.Body.Close()

	if wipe.StatusCode == http.StatusOK && len(actions) == 0 {
		t.Errorf("PM7 open bug: PUT /permissions/roles/writer/actions with a body missing the "+
			"declared-required %q field answered %d %s and wiped the role's actions to %v; "+
			"a missing required field should be refused (the operation declares \"actions\" "+
			"required: true), not silently treated as an explicit empty list", "actions",
			wipe.StatusCode, wipeBody, actions)
	}

	role := do("GET", "/permissions/roles/writer", "")
	var got permissions.Role
	if err := json.NewDecoder(role.Body).Decode(&got); err != nil {
		t.Fatalf("decode role: %v", err)
	}
	role.Body.Close()
	if got.Seeded {
		t.Errorf("PM7 open bug: the same missing-body PUT also marked the role seeded=true, "+
			"which permanently locks it out of AddAction/RemoveAction (ErrSeededRole) — a caller "+
			"cannot even hand-repair the accidental wipe")
	}
}
