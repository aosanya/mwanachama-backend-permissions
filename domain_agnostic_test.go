package mwanachamapermissions_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	permissions "github.com/aosanya/mwanachama-backend-permissions"
)

// domainWords are words that mean something in one domain and nothing in
// another. This module is expected to hold a hospital's clinical privileges
// as readily as a studio's operator roles, so none of them may appear in an
// identifier, a stored value or the blueprint.
//
// They are allowed in comments, where they earn their place by illustrating
// what a neutral name stands for. Note what is deliberately absent: "role",
// "grant", "action" and "scope" are this module's own vocabulary, general
// enough to mean the same thing everywhere, which is the test the rule
// actually applies.
var domainWords = []string{
	"agency", "agencies", "studio", "operator", "catalog", "archetype",
	"ward", "clinical", "practitioner", "patient", "nurse",
	"tenant", "employee", "member",
}

func carriesDomainWord(name string) (string, bool) {
	lower := strings.ToLower(name)
	for _, w := range domainWords {
		if strings.Contains(lower, w) {
			return w, true
		}
	}
	return "", false
}

func TestNoDomainWordsInIdentifiers(t *testing.T) {
	for _, dir := range []string{"models", "routes", "."} {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		seen := map[string]bool{}
		for _, pkg := range pkgs {
			for path, file := range pkg.Files {
				ast.Inspect(file, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if !ok {
						return true
					}
					w, bad := carriesDomainWord(id.Name)
					if !bad {
						return true
					}
					key := filepath.Base(path) + ":" + id.Name
					if seen[key] {
						return true
					}
					seen[key] = true
					t.Errorf("%s: identifier %q carries the domain word %q — a permission names an action, and what the action means belongs to whoever declared it",
						filepath.Base(path), id.Name, w)
					return true
				})
			}
		}
	}
}

func TestNoDomainWordsInTheBlueprint(t *testing.T) {
	b, err := permissions.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	report := func(kind, name string) {
		if w, bad := carriesDomainWord(name); bad {
			t.Errorf("blueprint %s %q carries the domain word %q — it belongs in a domain's own spec", kind, name, w)
		}
	}
	for _, o := range b.Objects {
		report("role", o.Role)
		if o.Name != "" || o.Table != "" {
			t.Errorf("blueprint role %q names a table, which is the domain's to choose", o.Role)
		}
		for _, f := range o.Fields {
			report("field", o.Role+"."+f.Name)
			for _, v := range f.Values {
				report("stored value", o.Role+"."+f.Name+"="+v)
			}
		}
		for _, idx := range o.Indexes {
			report("index", o.Role+"."+idx.Name)
		}
	}
}

// The module claims to be domain-agnostic; two domains that disagree about
// every noun are how that claim is tested rather than asserted.
func TestBothExampleDomainsLoadThroughTheBlueprint(t *testing.T) {
	for _, tc := range []struct {
		path  string
		entry string
		table string
	}{
		{"spec/examples/wakala.permissions.json", "role", "roles"},
		{"spec/examples/hospital.permissions.json", "privilege_set", "privilege_sets"},
	} {
		s, err := permissions.LoadSpec(tc.path)
		if err != nil {
			t.Fatalf("LoadSpec(%s): %v", tc.path, err)
		}
		o, ok := s.ByRole("role")
		if !ok {
			t.Fatalf("%s declares no object filling the role %q", tc.path, "role")
		}
		if o.Name != tc.entry || o.Table != tc.table {
			t.Errorf("%s: role %q is named %q in %q, want %q in %q",
				tc.path, "role", o.Name, o.Table, tc.entry, tc.table)
		}
	}
}

func TestTheTwoDomainsProduceDifferentTables(t *testing.T) {
	wakala, err := permissions.LoadSpec("spec/examples/wakala.permissions.json")
	if err != nil {
		t.Fatalf("LoadSpec(wakala): %v", err)
	}
	hospital, err := permissions.LoadSpec("spec/examples/hospital.permissions.json")
	if err != nil {
		t.Fatalf("LoadSpec(hospital): %v", err)
	}
	for _, role := range []string{"role", "role_action", "grant"} {
		a, _ := wakala.ByRole(role)
		b, _ := hospital.ByRole(role)
		if wakala.TableFor(a) == hospital.TableFor(b) {
			t.Errorf("role %q lands in the same table for both domains (%s) — the example is not demonstrating anything",
				role, wakala.TableFor(a))
		}
	}
}
