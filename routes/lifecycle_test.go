package routes_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

func TestRoutes_FullLifecycleThroughARealServer(t *testing.T) {
	srv := newProbeServer(t)
	client := srv.Client()

	do := func(method, path, body string) (int, []byte) {
		t.Helper()
		var reader io.Reader
		if body != "" {
			reader = bytes.NewBufferString(body)
		}
		req, err := http.NewRequest(method, srv.URL+path, reader)
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
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, b
	}

	if code, body := do("POST", "/permissions/roles", `{"code":"writer","name":"Writer"}`); code != http.StatusCreated {
		t.Fatalf("create_role: got %d %s", code, body)
	}
	if code, body := do("GET", "/permissions/roles/writer", ""); code != http.StatusOK {
		t.Fatalf("get_role: got %d %s", code, body)
	} else {
		var r permissions.Role
		if err := json.Unmarshal(body, &r); err != nil || r.Code != "writer" {
			t.Fatalf("get_role: decoded %s, err %v", body, err)
		}
	}
	if code, body := do("GET", "/permissions/roles", ""); code != http.StatusOK || !strings.Contains(string(body), `"writer"`) {
		t.Fatalf("list_roles: got %d %s", code, body)
	}
	if code, body := do("PATCH", "/permissions/roles/writer", `{"code":"writer","name":"Writer Renamed"}`); code != http.StatusOK {
		t.Fatalf("update_role: got %d %s", code, body)
	} else if !strings.Contains(string(body), "Writer Renamed") {
		t.Fatalf("update_role: name did not change: %s", body)
	}

	if code, body := do("POST", "/permissions/roles/writer/actions", `{"action":"catalog.entry.upsert"}`); code != http.StatusCreated {
		t.Fatalf("add_action: got %d %s", code, body)
	}
	if code, body := do("GET", "/permissions/roles/writer/actions", ""); code != http.StatusOK || !strings.Contains(string(body), "catalog.entry.upsert") {
		t.Fatalf("list_actions: got %d %s", code, body)
	}
	if code, body := do("DELETE", "/permissions/roles/writer/actions/catalog.entry.upsert", ""); code != http.StatusNoContent {
		t.Fatalf("remove_action: got %d %s", code, body)
	}
	if code, body := do("PUT", "/permissions/roles/writer/actions", `{"actions":["catalog.entry.upsert","catalog.entry.delete"]}`); code != http.StatusOK {
		t.Fatalf("set_actions: got %d %s", code, body)
	} else if !strings.Contains(string(body), "catalog.entry.delete") {
		t.Fatalf("set_actions: did not replace the list: %s", body)
	}

	if code, body := do("POST", "/permissions/grants",
		`{"subject":"alice","scope":"module:catalog","role_code":"writer"}`); code != http.StatusCreated {
		t.Fatalf("create_grant: got %d %s", code, body)
	}
	if code, body := do("GET", "/permissions/grants?subject=alice", ""); code != http.StatusOK || !strings.Contains(string(body), "alice") {
		t.Fatalf("list_grants: got %d %s", code, body)
	}
	if code, body := do("GET",
		"/permissions/check?subject=alice&scope=module:catalog&action=catalog.entry.upsert", ""); code != http.StatusOK || strings.TrimSpace(string(body)) != "true" {
		t.Fatalf("may_perform: got %d %s, want 200 true", code, body)
	}
	if code, body := do("GET", "/permissions/subjects/alice/actions?scope=module:catalog", ""); code != http.StatusOK ||
		!strings.Contains(string(body), "catalog.entry.upsert") || !strings.Contains(string(body), "catalog.entry.delete") {
		t.Fatalf("actions_for: got %d %s", code, body)
	}

	if code, body := do("DELETE", "/permissions/grants/alice/module:catalog/writer", ""); code != http.StatusNoContent {
		t.Fatalf("withdraw_grant: got %d %s", code, body)
	}
	if code, body := do("GET",
		"/permissions/check?subject=alice&scope=module:catalog&action=catalog.entry.upsert", ""); code != http.StatusOK || strings.TrimSpace(string(body)) != "false" {
		t.Fatalf("may_perform after withdraw: got %d %s, want 200 false", code, body)
	}

	if code, body := do("DELETE", "/permissions/roles/writer", ""); code != http.StatusNoContent {
		t.Fatalf("delete_role: got %d %s", code, body)
	}
	if code, body := do("GET", "/permissions/roles", ""); code != http.StatusOK || strings.Contains(string(body), `"writer"`) {
		t.Fatalf("list_roles after delete: got %d %s, want the retired role excluded", code, body)
	}
}
