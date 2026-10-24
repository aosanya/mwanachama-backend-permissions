package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAPartitionsPowersListThroughTheRouteTable(t *testing.T) {
	mux := mountPermissionsAs(t, "owner-1")
	for _, path := range []string{"/permissions/powers/agy1/run_poll/treasurer", "/permissions/powers/agy1/post_notice/chair"} {
		if rec := send(t, mux, "PUT", path, `{"held":true}`); rec.Code != http.StatusOK {
			t.Fatalf("override %s: %d %s", path, rec.Code, rec.Body)
		}
	}
	if rec := send(t, mux, "PUT", "/permissions/powers/agy2/run_poll/chair", `{"held":true}`); rec.Code != http.StatusOK {
		t.Fatalf("other partition: %d %s", rec.Code, rec.Body)
	}
	rec := send(t, mux, "GET", "/permissions/powers/agy1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	var got []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	if len(got) != 2 || got[0]["power"] != "post_notice" || got[1]["power"] != "run_poll" {
		t.Fatalf("listed %v; want agy1's two powers in order", got)
	}
	for _, p := range got {
		if p["partition"] != "agy1" || p["layer"] != "override" || p["set_by"] != "owner-1" || p["set_at"] == "" {
			t.Fatalf("row %v; want agy1's override set by the caller", p)
		}
	}
}
