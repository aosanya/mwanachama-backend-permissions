package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestListRolesNeverSurfacesARetiredRoleWhateverTheQuerySays(t *testing.T) {
	mux := mountPermissions(t)
	for _, code := range []string{"writer", "gone"} {
		if rec := send(t, mux, "POST", "/permissions/roles", `{"code":"`+code+`","name":"R"}`); rec.Code != http.StatusCreated {
			t.Fatalf("precondition: creating role %s answered %d %s, want 201", code, rec.Code, rec.Body.String())
		}
	}
	if rec := send(t, mux, "DELETE", "/permissions/roles/gone", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("precondition: retiring gone answered %d %s, want 204", rec.Code, rec.Body.String())
	}
	if rec := send(t, mux, "GET", "/permissions/roles/gone", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"deleted":true`) {
		t.Fatalf("precondition: gone should read back retired, got %d %s", rec.Code, rec.Body.String())
	}

	for _, query := range []string{
		"",
		"?code=gone",
		"?code=GONE",
		"?code=g%25",
		"?name=R",
		"?deleted=true",
		"?Deleted=true",
		"?seeded=false",
		"?seeded=null",
		"?code=writer&code=gone",
		"?code=gone&code=writer",
	} {
		rec := send(t, mux, "GET", "/permissions/roles"+query, "")
		if rec.Code != http.StatusOK {
			t.Errorf("GET /permissions/roles%s answered %d %s, want 200", query, rec.Code, rec.Body.String())
			continue
		}
		if strings.Contains(rec.Body.String(), `"gone"`) || strings.Contains(rec.Body.String(), `"deleted":true`) {
			t.Errorf("GET /permissions/roles%s listed the retired role: %s — a listing carries live roles only, whatever the query asks for", query, rec.Body.String())
		}
	}
}
