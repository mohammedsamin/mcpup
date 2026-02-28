package registry

import (
	"slices"
	"testing"
)

func TestAllSortedByName(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatalf("expected non-empty registry")
	}

	names := make([]string, 0, len(all))
	for _, tmpl := range all {
		names = append(names, tmpl.Name)
	}
	if !slices.IsSorted(names) {
		t.Fatalf("expected templates to be sorted by name")
	}
}

func TestLookupCaseInsensitive(t *testing.T) {
	tmpl, ok := Lookup("GITHUB")
	if !ok {
		t.Fatalf("expected github template to exist")
	}
	if tmpl.Name != "github" {
		t.Fatalf("expected github template, got %q", tmpl.Name)
	}
}

func TestSearchMatchesCategory(t *testing.T) {
	results := Search("database")
	if len(results) == 0 {
		t.Fatalf("expected category search to return results")
	}

	foundPostgres := false
	for _, tmpl := range results {
		if tmpl.Name == "postgres" {
			foundPostgres = true
			break
		}
	}
	if !foundPostgres {
		t.Fatalf("expected postgres template in database search results")
	}
}

func TestCategoriesSortedUnique(t *testing.T) {
	cats := Categories()
	if len(cats) == 0 {
		t.Fatalf("expected categories")
	}
	if !slices.IsSorted(cats) {
		t.Fatalf("expected categories sorted")
	}

	seen := map[string]struct{}{}
	for _, cat := range cats {
		if _, exists := seen[cat]; exists {
			t.Fatalf("duplicate category %q", cat)
		}
		seen[cat] = struct{}{}
	}
}
